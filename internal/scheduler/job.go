package scheduler

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/navisha/spark/internal/config"
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
	categoryConfigs  []config.Category
	log              *logrus.Entry
	rng              *rand.Rand
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
	categoryConfigs []config.Category,
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
		categoryConfigs:  categoryConfigs,
		log:              log,
		rng:              rand.New(rand.NewSource(time.Now().UnixNano())),
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

	if selection != nil {
		category = selection.Category
		level = selection.Level

		if selection.FromBank {
			// Use insight directly from bank
			insight = selection.Insight
			j.log.WithFields(logrus.Fields{
				"insight_id": insight.ID,
				"category":   category,
				"level":      level,
			}).Info("using insight from bank")
		} else if selection.Insight != nil && selection.Key != "" {
			// Generate variation of existing insight with the same key
			j.log.WithFields(logrus.Fields{
				"base_id":  selection.Insight.ID,
				"category": category,
				"level":    level,
				"key":      selection.Key,
			}).Info("generating variation of existing insight")

			generated, err := j.contentGen.GenerateVariationForKey(
				ctx,
				category,
				level,
				selection.Key,
				selection.Insight.Insight,
			)
			if err != nil {
				return fmt.Errorf("failed to generate variation for key %s: %w", selection.Key, err)
			}

			// Validate generated content
			validationResult := j.contentValidator.Validate(generated)
			if !validationResult.Valid {
				j.log.WithFields(logrus.Fields{
					"errors":   validationResult.Errors,
					"warnings": validationResult.Warnings,
				}).Warn("generated variation has validation issues")
			}

			// Save variation to database
			insight = &repository.Insight{
				Category:  category,
				Level:     level,
				Title:     generated.Title,
				Insight:   generated.Insight,
				Key:       &selection.Key,
				KeyPoints: generated.KeyPoints,
				Tags:      generated.Tags,
			}

			if generated.CodeExample != "" {
				insight.CodeExample = &generated.CodeExample
			}

			if err := j.insightRepo.Create(ctx, insight); err != nil {
				return fmt.Errorf("failed to save variation: %w", err)
			}

			j.log.WithFields(logrus.Fields{
				"insight_id": insight.ID,
				"base_id":    selection.Insight.ID,
				"category":   category,
				"level":      level,
				"key":        selection.Key,
				"title":      insight.Title,
			}).Info("variation generated and saved")
		} else {
			// selection.FromBank is false but no key - treat as new insight needed
			j.log.Info("selection returned without bank insight or key, will generate new insight")
		}
	}

	// If no insight was set from bank or variation, generate new one
	if insight == nil {
		// Generate completely new insight via LLM
		j.log.Info("generating new insight via LLM")

		// Use rotation-based category selection instead of always using first category
		if len(j.categories) == 0 {
			return fmt.Errorf("no categories configured")
		}

		selectedCategory, err := j.rotationEngine.SelectCategoryForLLM(ctx, j.categories)
		if err != nil {
			j.log.WithError(err).Warn("failed to select category via rotation, using fallback")
			selectedCategory = j.categories[0]
		}
		category = selectedCategory

		// Select level using rotation engine's selector
		_, stateErr := j.rotationEngine.GetCategoryPriority(ctx, category)
		var rotationState *repository.RotationState
		if stateErr == nil {
			state, err := j.rotationEngine.GetCategoryPriority(ctx, category)
			if err == nil && state > 0 {
				states, getAllErr := j.rotationEngine.GetAllRotationStates(ctx)
				if getAllErr == nil {
					for i := range states {
						if states[i].Category == category {
							rotationState = &states[i]
							break
						}
					}
				}
			}
		}

		selector := rotation.NewSelector(rotation.EngineConfig{
			LevelDistribution:   map[string]int{"beginner": 20, "intermediate": 50, "advanced": 30},
			MinDaysBeforeRepeat: 7,
			DedupWindowHours:    24,
			DefaultWeight:       1.0,
		})
		level = selector.SelectLevel(rotationState)

		// Pick a random subtopic/key from the category config
		key := j.pickRandomSubtopic(category)

		j.log.WithFields(logrus.Fields{
			"category": category,
			"level":    level,
			"key":      key,
		}).Info("selected category and key for LLM generation")

		// Generate insight using the key for better prompt variation
		generated, err := j.contentGen.GenerateInsightWithKey(ctx, category, level, key)
		if err != nil {
			return fmt.Errorf("failed to generate insight with key %s: %w", key, err)
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
			Key:       &key,
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

	messages := j.telegramFmt.Format(msgData)

	// Step 3: Send via Telegram with retry
	var lastMsgID int64
	for i, msg := range messages {
		msgID, err := j.telegramClient.SendMessage(ctx, msg)
		if err != nil {
			return fmt.Errorf("failed to send message part %d/%d: %w", i+1, len(messages), err)
		}
		lastMsgID = msgID
		j.log.WithFields(logrus.Fields{
			"message_id":  msgID,
			"part":        i + 1,
			"total_parts": len(messages),
			"insight_id":  insight.ID,
		}).Info("message part sent successfully")
	}

	// Step 4: Record delivery
	if err := j.rotationEngine.RecordDelivery(ctx, category, level, insight.ID); err != nil {
		j.log.WithError(err).Error("failed to record delivery")
	}

	j.log.WithFields(logrus.Fields{
		"insight_id":  insight.ID,
		"total_parts": len(messages),
		"last_msg_id": lastMsgID,
	}).Info("all message parts sent successfully")

	return nil
}

// pickRandomSubtopic picks a random subtopic from the category configuration.
func (j *SendInsightJob) pickRandomSubtopic(categoryName string) string {
	for _, cat := range j.categoryConfigs {
		if cat.Name == categoryName && len(cat.Subtopics) > 0 {
			idx := j.rng.Intn(len(cat.Subtopics))
			return cat.Subtopics[idx]
		}
	}
	return "general"
}

// ExecuteWithTimeout runs the job with a timeout.
func (j *SendInsightJob) ExecuteWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return j.Execute(ctx)
}
