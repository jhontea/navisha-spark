package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"

	"github.com/navisha/spark/internal/config"
	"github.com/navisha/spark/internal/content"
	"github.com/navisha/spark/internal/database"
	"github.com/navisha/spark/internal/database/repository"
	"github.com/navisha/spark/internal/rotation"
	"github.com/navisha/spark/internal/scheduler"
	"github.com/navisha/spark/internal/telegram"
)

func main() {
	// Load .env file (warn only; env vars may be set directly in production)
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Warning: failed to load .env file: %v\n", err)
	}

	// Load configuration
	cfg := config.MustConfig("config/config.yaml")

	// Setup logger
	logger := setupLogger(cfg)
	logger.WithFields(logrus.Fields{
		"app":     cfg.App.Name,
		"env":     cfg.App.Env,
		"version": "1.0.0",
	}).Info("starting Navisha Spark")

	// Create root context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database
	db, err := database.New(config.DatabaseConfig{
		URL:             cfg.Database.URL,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
	}, logger)
	if err != nil {
		logger.WithError(err).Fatal("failed to connect to database")
	}
	defer db.Close()
	logger.Info("database connection established")

	// Initialize repositories
	insightRepo := repository.NewInsightRepository(db.DB)
	rotationRepo := repository.NewRotationRepository(db.DB)
	historyRepo := repository.NewHistoryRepository(db.DB)

	// Initialize Telegram client
	telegramCfg := telegram.Config{
		BotToken:              cfg.Telegram.BotToken,
		ChatID:                cfg.Telegram.ChatID,
		ParseMode:             cfg.Telegram.ParseMode,
		DisableWebPagePreview: cfg.Telegram.DisableWebPagePreview,
		DisableNotification:   cfg.Telegram.DisableNotification,
	}
	telegramClient, err := telegram.NewClient(telegramCfg, logger)
	if err != nil {
		logger.WithError(err).Fatal("failed to initialize Telegram client")
	}
	logger.Info("Telegram client initialized")

	// Initialize Telegram formatter
	telegramFmt := telegram.NewFormatter(telegram.FormatConfig{
		IncludeCategory:  cfg.Format.IncludeCategory,
		IncludeLevel:     cfg.Format.IncludeLevel,
		IncludeFollowUps: cfg.Format.IncludeFollowUps,
		IncludeTags:      cfg.Format.IncludeTags,
		MarkdownEnabled:  cfg.Format.MarkdownEnabled,
	})

	// Initialize content generator
	contentGen := content.NewGenerator(
		cfg.LLM.APIKey,
		content.PromptConfig{
			Model:       cfg.LLM.Model,
			MaxTokens:   cfg.LLM.MaxTokens,
			Temperature: cfg.LLM.Temperature,
		},
		logger,
	)

	// Initialize content validator
	contentValidator := content.NewValidator()

	// Initialize rotation engine
	rotationEngine := rotation.NewEngine(
		rotationRepo,
		insightRepo,
		historyRepo,
		rotation.EngineConfig{
			LevelDistribution:   cfg.Rotation.LevelDistribution,
			MinDaysBeforeRepeat: cfg.Rotation.MinDaysBeforeRepeat,
			DedupWindowHours:    cfg.Deduplication.WindowHours,
			DefaultWeight:       cfg.Rotation.WeightedRoundRobin.DefaultWeight,
		},
		logger,
	)

	// Initialize scheduler
	schedulerCfg := scheduler.Config{
		Cron:        cfg.Schedule.Cron,
		Timezone:    cfg.Schedule.Timezone,
		ActiveStart: cfg.Schedule.ActiveHours.Start,
		ActiveEnd:   cfg.Schedule.ActiveHours.End,
	}
	sched, err := scheduler.New(schedulerCfg, logger)
	if err != nil {
		logger.WithError(err).Fatal("failed to initialize scheduler")
	}

	// Create send insight job
	categories := getEnabledCategories(cfg.Categories)
	sendInsightJob := scheduler.NewSendInsightJob(
		rotationEngine,
		insightRepo,
		contentGen,
		contentValidator,
		telegramClient,
		telegramFmt,
		categories,
		cfg.Categories,
		logger,
	)

	// Add job to scheduler
	if err := sched.AddJob("send_insight", cfg.Schedule.Cron, func() {
		if err := sendInsightJob.ExecuteWithTimeout(2 * time.Minute); err != nil {
			logger.WithError(err).Error("insight delivery job failed")
		}
	}); err != nil {
		logger.WithError(err).Fatal("failed to add job to scheduler")
	}

	// Start scheduler
	sched.Start()
	logger.Info("scheduler started")

	// Setup HTTP server with all endpoints
	healthServer := setupHealthServer(db.DB, telegramClient, logger, cfg.App.Port, sendInsightJob)

	// Start HTTP server in background
	go func() {
		logger.WithField("port", cfg.App.Port).Info("health check server starting")
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("health check server failed")
		}
	}()

	// Setup config watcher for hot-reload
	if err := config.WatchConfig("config/config.yaml", func(newCfg *config.Config) {
		logger.Info("configuration reloaded")

		// Update scheduler with new cron expression if it changed
		oldCron := cfg.Schedule.Cron
		newCron := newCfg.Schedule.Cron

		if oldCron != newCron {
			logger.WithFields(logrus.Fields{
				"old_cron": oldCron,
				"new_cron": newCron,
			}).Info("updating scheduler cron expression")

			if err := sched.RemoveJob("send_insight"); err != nil {
				logger.WithError(err).Error("failed to remove old job")
				return
			}

			if err := sched.AddJob("send_insight", newCron, func() {
				if err := sendInsightJob.ExecuteWithTimeout(2 * time.Minute); err != nil {
					logger.WithError(err).Error("insight delivery job failed")
				}
			}); err != nil {
				logger.WithError(err).Error("failed to add job with new cron")
				return
			}

			logger.Info("scheduler cron expression updated successfully")
		}

		cfg = newCfg
	}); err != nil {
		logger.WithError(err).Warn("failed to setup config watcher")
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("application started successfully, waiting for shutdown signal")
	<-quit

	logger.Info("shutting down application...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 30*time.Second)
	defer shutdownCancel()

	// Stop scheduler
	if schedCtx := sched.Stop(); schedCtx != nil {
		<-schedCtx.Done()
	}
	logger.Info("scheduler stopped")

	// Shutdown HTTP server
	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("error shutting down health server")
	} else {
		logger.Info("health server stopped")
	}

	logger.Info("application shutdown complete")
}

// setupLogger configures the global logger.
func setupLogger(cfg *config.Config) *logrus.Entry {
	logger := logrus.New()

	level, err := logrus.ParseLevel(cfg.App.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	if cfg.Logging.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	}

	return logger.WithField("app", cfg.App.Name)
}

// setupHealthServer creates the HTTP server for health checks and manual trigger.
func setupHealthServer(
	db *sqlx.DB,
	telegramClient *telegram.Client,
	logger *logrus.Entry,
	port int,
	sendInsightJob *scheduler.SendInsightJob,
) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		status := "ok"
		dbStatus := "healthy"
		tgStatus := "healthy"

		if err := db.Ping(); err != nil {
			logger.WithError(err).Error("health check: database ping failed")
			status = "degraded"
			dbStatus = "unhealthy"
		}

		checkCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if err := telegramClient.HealthCheck(checkCtx); err != nil {
			logger.WithError(err).Warn("health check: Telegram API unreachable")
			tgStatus = "unhealthy"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":%q,"timestamp":%q,"database":%q,"telegram":%q}`,
			status,
			time.Now().UTC().Format(time.RFC3339),
			dbStatus,
			tgStatus,
		)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Navisha Spark is running")
	})

	// /trigger accepts GET only; protected by Nginx rate-limiting in production.
	// For local/dev use, combine with basic auth or IP restriction as needed.
	mux.HandleFunc("/trigger", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		logger.Info("manual trigger received")

		go func() {
			if err := sendInsightJob.ExecuteWithTimeout(2 * time.Minute); err != nil {
				logger.WithError(err).Error("manual trigger failed")
			} else {
				logger.Info("manual trigger completed successfully")
			}
		}()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprint(w, `{"status":"accepted","message":"insight delivery started"}`)
	})

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}
}

// parseDelays converts string delays to time.Duration.
func parseDelays(delays []string) []time.Duration {
	result := make([]time.Duration, len(delays))
	for i, d := range delays {
		duration, err := time.ParseDuration(d)
		if err != nil {
			duration = time.Minute
		}
		result[i] = duration
	}
	return result
}

// getEnabledCategories returns names of all enabled categories.
func getEnabledCategories(categories []config.Category) []string {
	var enabled []string
	for _, cat := range categories {
		if cat.Enabled {
			enabled = append(enabled, cat.Name)
		}
	}
	return enabled
}
