package config

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

// Global config pointer for atomic hot-reload
var globalConfig atomic.Value

// LoadConfig loads configuration from config.yaml with env overrides.
func LoadConfig(configPath string) (*Config, error) {
	cfg := &Config{}
	setDefaults(cfg)

	// Load from YAML
	if err := loadFromYAML(configPath, cfg); err != nil {
		return nil, fmt.Errorf("failed to load YAML config: %w", err)
	}

	// Override with environment variables
	loadFromEnv(cfg)

	// Store atomically for hot-reload
	globalConfig.Store(cfg)

	return cfg, nil
}

// GetConfig returns the current config (thread-safe, atomic read).
func GetConfig() *Config {
	if v := globalConfig.Load(); v != nil {
		return v.(*Config)
	}
	return nil
}

// WatchConfig watches config.yaml for changes and hot-reloads.
func WatchConfig(configPath string, onChange func(*Config)) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	if err := watcher.Add(configPath); err != nil {
		return fmt.Errorf("failed to watch %s: %w", configPath, err)
	}

	go func() {
		var debounceTimer *time.Timer
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					// Debounce: wait 100ms after last write
					if debounceTimer != nil {
						debounceTimer.Stop()
					}
					debounceTimer = time.AfterFunc(100*time.Millisecond, func() {
						cfg, err := LoadConfig(configPath)
						if err != nil {
							fmt.Printf("failed to reload config: %v\n", err)
							return
						}
						onChange(cfg)
					})
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	return nil
}

// loadFromYAML loads configuration from YAML file.
func loadFromYAML(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	return nil
}

// loadFromEnv overrides config with environment variables.
func loadFromEnv(cfg *Config) {
	if v := os.Getenv("APP_NAME"); v != "" {
		cfg.App.Name = v
	}
	if v := os.Getenv("APP_ENV"); v != "" {
		cfg.App.Env = v
	}
	if v := os.Getenv("APP_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.App.Port)
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.App.LogLevel = v
	}

	if v := os.Getenv("TELEGRAM_BOT_TOKEN"); v != "" {
		cfg.Telegram.BotToken = v
	}
	if v := os.Getenv("TELEGRAM_CHAT_ID"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.Telegram.ChatID)
	}

	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.Database.URL = v
	}

	if v := os.Getenv("OPENROUTER_API_KEY"); v != "" {
		cfg.LLM.APIKey = v
	}
	if v := os.Getenv("OPENROUTER_MODEL"); v != "" {
		cfg.LLM.Model = v
	}
	if v := os.Getenv("LLM_MAX_TOKENS"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.LLM.MaxTokens)
	}
	if v := os.Getenv("LLM_TEMPERATURE"); v != "" {
		fmt.Sscanf(v, "%f", &cfg.LLM.Temperature)
	}

	if v := os.Getenv("SCHEDULE_CRON"); v != "" {
		cfg.Schedule.Cron = v
	}
	if v := os.Getenv("TIMEZONE"); v != "" {
		cfg.Schedule.Timezone = v
	}
	if v := os.Getenv("DEDUP_WINDOW_HOURS"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.Deduplication.WindowHours)
	}
	if v := os.Getenv("MAX_RETRIES"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.Retry.MaxRetries)
	}
}

// setDefaults sets default configuration values.
func setDefaults(cfg *Config) {
	cfg.App.Name = "navisha-spark"
	cfg.App.Env = "production"
	cfg.App.Port = 8080
	cfg.App.LogLevel = "info"

	cfg.Database.MaxOpenConns = 10
	cfg.Database.MaxIdleConns = 5
	cfg.Database.ConnMaxLifetime = 5 * time.Minute

	cfg.LLM.Model = "openrouter/owl-alpha"
	cfg.LLM.MaxTokens = 1000
	cfg.LLM.Temperature = 0.7

	cfg.Schedule.Cron = "0 */3 * * *"
	cfg.Schedule.Timezone = "Asia/Jakarta"
	cfg.Schedule.ActiveHours.Start = 0
	cfg.Schedule.ActiveHours.End = 23
	cfg.Rotation.MinDaysBeforeRepeat = 7
	cfg.Deduplication.WindowHours = 24

	cfg.Retry.MaxRetries = 3
	cfg.Retry.Delays = []string{"1m", "5m", "15m"}

	cfg.Format.IncludeCategory = true
	cfg.Format.IncludeLevel = true
	cfg.Format.IncludeFollowUps = true
	cfg.Format.IncludeTags = true
	cfg.Format.MarkdownEnabled = true

	cfg.Logging.Level = "info"
	cfg.Logging.Format = "json"
}
