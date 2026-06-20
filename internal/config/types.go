// Package config provides configuration management for Navisha Spark.
// Supports hot-reload: edit config.yaml and changes reflect immediately.
package config

import "time"

// Config is the main application configuration (from config.yaml + env overrides).
type Config struct {
	App           AppConfig      `yaml:"app"`
	Telegram      TelegramConfig `yaml:"telegram"`
	Database      DatabaseConfig `yaml:"database"`
	LLM           LLMConfig      `yaml:"llm"`
	Schedule      ScheduleConfig `yaml:"schedule"`
	Rotation      RotationConfig `yaml:"rotation"`
	Deduplication DedupConfig    `yaml:"deduplication"`
	Retry         RetryConfig    `yaml:"retry"`
	Format        FormatConfig   `yaml:"format"`
	Logging       LoggingConfig  `yaml:"logging"`
	Categories    []Category     `yaml:"categories"`
}

// AppConfig holds application-level settings.
type AppConfig struct {
	Name     string `yaml:"name"`
	Env      string `yaml:"env"`
	Port     int    `yaml:"port"`
	LogLevel string `yaml:"log_level"`
}

// TelegramConfig holds Telegram bot settings.
type TelegramConfig struct {
	BotToken              string `yaml:"bot_token"`
	ChatID                int64  `yaml:"chat_id"`
	ParseMode             string `yaml:"parse_mode"`
	DisableWebPagePreview bool   `yaml:"disable_web_page_preview"`
	DisableNotification   bool   `yaml:"disable_notification"`
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	URL             string        `yaml:"url"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// LLMConfig holds LLM/content generation settings.
type LLMConfig struct {
	APIKey         string  `yaml:"api_key"`
	Model          string  `yaml:"model"`
	MaxTokens      int     `yaml:"max_tokens"`
	Temperature    float64 `yaml:"temperature"`
	PrimarySource  string  `yaml:"primary_source"`
	FallbackSource string  `yaml:"fallback_source"`
}

// ScheduleConfig holds scheduling settings.
type ScheduleConfig struct {
	Cron        string      `yaml:"cron"`
	Timezone    string      `yaml:"timezone"`
	ActiveHours ActiveHours `yaml:"active_hours"`
}

// ActiveHours defines when messages should be sent.
type ActiveHours struct {
	Start int `yaml:"start"`
	End   int `yaml:"end"`
}

// RotationConfig holds topic rotation settings.
type RotationConfig struct {
	LevelDistribution   map[string]int           `yaml:"level_distribution"`
	MinDaysBeforeRepeat int                      `yaml:"min_days_before_repeat"`
	WeightedRoundRobin  WeightedRoundRobinConfig `yaml:"weighted_round_robin"`
}

// WeightedRoundRobinConfig holds weighted round-robin settings.
type WeightedRoundRobinConfig struct {
	Enabled       bool    `yaml:"enabled"`
	DefaultWeight float64 `yaml:"default_weight"`
}

// DedupConfig holds deduplication settings.
type DedupConfig struct {
	WindowHours int `yaml:"window_hours"`
}

// RetryConfig holds retry settings.
type RetryConfig struct {
	MaxRetries int      `yaml:"max_retries"`
	Delays     []string `yaml:"delays"`
}

// FormatConfig holds message format settings.
type FormatConfig struct {
	IncludeCategory  bool `yaml:"include_category"`
	IncludeLevel     bool `yaml:"include_level"`
	IncludeFollowUps bool `yaml:"include_follow_ups"`
	IncludeTags      bool `yaml:"include_tags"`
	MarkdownEnabled  bool `yaml:"markdown_enabled"`
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// Category represents a topic category.
type Category struct {
	Name      string   `yaml:"name"`
	Enabled   bool     `yaml:"enabled"`
	Weight    float64  `yaml:"weight"`
	Subtopics []string `yaml:"subtopics"`
}
