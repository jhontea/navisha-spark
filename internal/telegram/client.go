package telegram

import (
	"context"
	"fmt"
	"net/http"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sirupsen/logrus"
)

// Client wraps the Telegram Bot API client.
type Client struct {
	api    *tgbotapi.BotAPI
	chatID int64
	config Config
	log    *logrus.Entry
	client *http.Client
}

// Config holds Telegram client configuration.
type Config struct {
	BotToken              string
	ChatID                int64
	ParseMode             string
	DisableWebPagePreview bool
	DisableNotification   bool
}

// NewClient creates a new Telegram client.
func NewClient(cfg Config, log *logrus.Entry) (*Client, error) {
	api, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram bot: %w", err)
	}

	api.Debug = false

	log.WithFields(logrus.Fields{
		"bot_name": api.Self.UserName,
		"chat_id":  cfg.ChatID,
	}).Info("telegram bot initialized")

	return &Client{
		api:    api,
		chatID: cfg.ChatID,
		config: cfg,
		log:    log,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// SendMessage sends a text message to the configured chat.
func (c *Client) SendMessage(ctx context.Context, text string) (int64, error) {
	msg := tgbotapi.NewMessage(c.chatID, text)
	msg.ParseMode = c.config.ParseMode
	msg.DisableWebPagePreview = c.config.DisableWebPagePreview
	msg.DisableNotification = c.config.DisableNotification

	resp, err := c.api.Send(msg)
	if err != nil {
		return 0, fmt.Errorf("failed to send message: %w", err)
	}

	c.log.WithFields(logrus.Fields{
		"message_id": resp.MessageID,
		"chat_id":    c.chatID,
	}).Debug("message sent successfully")

	return int64(resp.MessageID), nil
}

// SendMessageToChat sends a message to a specific chat ID.
func (c *Client) SendMessageToChat(ctx context.Context, chatID int64, text string) (int64, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = c.config.ParseMode
	msg.DisableWebPagePreview = c.config.DisableWebPagePreview
	msg.DisableNotification = c.config.DisableNotification

	resp, err := c.api.Send(msg)
	if err != nil {
		return 0, fmt.Errorf("failed to send message to chat %d: %w", chatID, err)
	}

	c.log.WithFields(logrus.Fields{
		"message_id": resp.MessageID,
		"chat_id":    chatID,
	}).Debug("message sent to specific chat")

	return int64(resp.MessageID), nil
}

// SendMessageWithRetry sends a message with retry logic.
func (c *Client) SendMessageWithRetry(ctx context.Context, text string, maxRetries int, delays []time.Duration) (int64, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := delays[attempt-1]
			c.log.WithFields(logrus.Fields{
				"attempt": attempt,
				"delay":   delay.String(),
			}).Debug("retrying message send")

			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(delay):
			}
		}

		msgID, err := c.SendMessage(ctx, text)
		if err == nil {
			return msgID, nil
		}

		lastErr = err
		c.log.WithFields(logrus.Fields{
			"attempt": attempt + 1,
			"error":   err.Error(),
		}).Warn("failed to send message")
	}

	return 0, fmt.Errorf("failed to send message after %d retries: %w", maxRetries, lastErr)
}

// GetMe checks if the bot token is valid.
func (c *Client) GetMe(ctx context.Context) error {
	_, err := c.api.GetMe()
	if err != nil {
		return fmt.Errorf("failed to get bot info: %w", err)
	}
	return nil
}

// HealthCheck checks if the Telegram API is reachable.
func (c *Client) HealthCheck(ctx context.Context) error {
	_, err := c.api.GetMe()
	return err
}

// GetUpdates retrieves recent updates (for debugging).
func (c *Client) GetUpdates(ctx context.Context) ([]tgbotapi.Update, error) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 5

	updates, err := c.api.GetUpdates(u)
	if err != nil {
		return nil, fmt.Errorf("failed to get updates: %w", err)
	}

	return updates, nil
}
