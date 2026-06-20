package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/navisha/spark/internal/content"
	"github.com/navisha/spark/internal/database/repository"
	"github.com/navisha/spark/internal/rotation"
	"github.com/navisha/spark/internal/telegram"
)

// SendInsightJob represents the job that sends an insight to Telegram.
type SendInsightJob struct {
	rotationEngine   *rotation.Engine
	insightRepo      *repository.InsightRepository
	contentGen       *content.Generator
	contentValidator *content.Validator
	telegramClient   *telegram.Client
	telegramFmt      *telegram.Formatter
	categories       []string
	log              *logrus.Entry
}

// NewSendInsightJob creates a new SendInsightJob.
func NewSendInsightJob(
	rotationEngine *rotation.Engine,
	insightRepo *repository.InsightRepository,
	contentGen *content.Generator,
	contentValidator *content.Validator,
	telegramClient *telegram.Client,
	telegramFmt *telegram.Formatter,
	categories []string,
	log *logrus.Entry,
) *SendInsightJob {
	return &SendInsightJob{
		rotationEngine:   rotationEngine,
		insightRepo:      insightRepo,
		contentGen:       contentGen,
		contentValidator: contentValidator,
		telegramClient:   telegramClient,
		telegramFmt:      telegramFmt,
		categories:       categories,
		log:              log,
	}
}

// Execute runs the job to select and send an insight.
func (j *SendInsightJob) Execute(ctx context.Context) error {
	j.log.Info("starting insight delivery job")

	// Step 1: Select next insight using rotation engine
	selection, err := j.rotationEngine.SelectNext(ctx, j.categories)
	if err != nil {
		return fmt.Errorf("failed to select next insight: %w", err)
	}

	var insight *repository.Insight
	var category, level string

	if selection != nil && selection.FromBank {
		// Use insight from bank
		insight = selection.Insight
		category = selection.Category
		level = selection.Level
		j.log.WithFields(logrus.Fields{
			"insight_id": insight.ID,
			"category":   category,
			"level":      level,
		}).Info("using insight from bank")
	} else {
		// Generate new insight via LLM
		j.log.Info("no insight in bank, generating via LLM")

		// Use first available category
		if len(j.categories) == 0 {
			return fmt.Errorf("no categories configured")
		}
		category = j.categories[0]
		level = "intermediate"

		generated, err := j.contentGen.GenerateInsight(ctx, category, level, "general")
		if err != nil {
			return fmt.Errorf("failed to generate insight: %w", err)
		}

		// Validate generated content
		validationResult := j.contentValidator.Validate(generated)
		if !validationResult.Valid {
			j.log.WithFields(logrus.Fields{
				"errors":   validationResult.Errors,
				"warnings": validationResult.Warnings,
			}).Warn("generated insight has validation issues")
		}

		// Save generated insight to database
		insight = &repository.Insight{
			Category:  category,
			Level:     level,
			Title:     generated.Title,
			Insight:   generated.Insight,
			KeyPoints: generated.KeyPoints,
			Tags:      generated.Tags,
		}

		if generated.CodeExample != "" {
			insight.CodeExample = &generated.CodeExample
		}

		if err := j.insightRepo.Create(ctx, insight); err != nil {
			return fmt.Errorf("failed to save generated insight: %w", err)
		}

		j.log.WithFields(logrus.Fields{
			"insight_id": insight.ID,
			"category":   category,
			"level":      level,
			"title":      insight.Title,
		}).Info("generated and saved new insight")
	}

	// Step 2: Format the message
	msgData := &telegram.InsightData{
		Category:  category,
		Level:     level,
		Title:     insight.Title,
		Insight:   insight.Insight,
		KeyPoints: insight.KeyPoints,
		Tags:      insight.Tags,
	}

	if insight.CodeExample != nil {
		msgData.CodeExample = *insight.CodeExample
	}

	message := j.telegramFmt.Format(msgData)

	// Step 3: Send via Telegram with retry
	msgID, err := j.telegramClient.SendMessage(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	j.log.WithFields(logrus.Fields{
		"message_id": msgID,
		"insight_id": insight.ID,
	}).Info("message sent successfully")

	// Step 4: Record delivery
	if err := j.rotationEngine.RecordDelivery(ctx, category, level, insight.ID); err != nil {
		j.log.WithError(err).Error("failed to record delivery")
		// Don't return error, message was already sent
	}

	return nil
}

// ExecuteWithTimeout runs the job with a timeout.
func (j *SendInsightJob) ExecuteWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return j.Execute(ctx)
}
