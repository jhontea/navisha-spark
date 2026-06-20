# Telegram Bot API — Skill Guide

## Overview

Skill ini berisi panduan lengkap untuk menggunakan Telegram Bot API dalam Navisha Spark. Mencakup setup, best practices, message formatting, error handling, dan common pitfalls.

---

## 1. Bot Setup

### 1.1 Create Bot via BotFather

1. Buka Telegram, cari **@BotFather**
2. Kirim command `/newbot`
3. Ikuti instruksi:
   - Masukkan nama bot (display name)
   - Masukkan username (harus unik, akhiran `bot`)
4. Simpan **Bot Token** yang diberikan (format: `123456789:ABCdef...`)

### 1.2 Get Chat ID

1. Buka bot yang baru dibuat
2. Kirim pesan apapun ke bot
3. Buka browser, akses:
   ```
   https://api.telegram.org/bot<TOKEN>/getUpdates
   ```
4. Cari `"chat":{"id":203294061` — angka tersebut adalah Chat ID

### 1.3 Set Bot Commands (Optional)

```go
commands := []tgbotapi.BotCommand{
    {
        Command:     "start",
        Description: "Start the bot",
    },
    {
        Command:     "help",
        Description: "Show help message",
    },
    {
        Command:     "status",
        Description: "Check bot status",
    },
}

_, err := bot.Request(tgbotapi.SetMyCommandsConfig{
    Commands: commands,
})
if err != nil {
    logrus.WithError(err).Error("Failed to set bot commands")
}
```

---

## 2. Basic Usage

### 2.1 Initialize Bot

```go
import "github.com/go-telegram-bot-api/telegram-bot-api/v6"

func NewTelegramClient(token string) (*TelegramClient, error) {
    bot, err := tgbotapi.NewBotAPI(token)
    if err != nil {
        return nil, fmt.Errorf("failed to create bot: %w", err)
    }

    bot.Debug = false // Set true untuk development
    logrus.WithField("username", bot.Self.UserName).Info("Bot authorized")

    return &TelegramClient{bot: bot}, nil
}
```

### 2.2 Send Simple Message

```go
func (c *TelegramClient) SendMessage(chatID int64, text string) error {
    msg := tgbotapi.NewMessage(chatID, text)
    _, err := c.bot.Send(msg)
    if err != nil {
        return fmt.Errorf("failed to send message: %w", err)
    }
    return nil
}
```

### 2.3 Send Message with Markdown

```go
func (c *TelegramClient) SendMarkdown(chatID int64, text string) error {
    msg := tgbotapi.NewMessage(chatID, text)
    msg.ParseMode = "Markdown"
    msg.DisableWebPagePreview = true
    
    _, err := c.bot.Send(msg)
    if err != nil {
        return fmt.Errorf("failed to send markdown message: %w", err)
    }
    return nil
}
```

---

## 3. Message Formatting

### 3.1 Markdown Formatting

Telegram Bot API mendukung dua format Markdown:
- `Markdown` (legacy, menggunakan `*bold*`, `_italic_`, `` `code` ``)
- `MarkdownV2` (baru, menggunakan `*bold*`, `_italic_`, `` `code` ``, `~strikethrough~`)

**Navisha Spark menggunakan `Markdown` untuk compatibility.**

### 3.2 Question Message Format

```go
func FormatQuestionMessage(q *Question) string {
    var sb strings.Builder

    // Header: Category + Level
    sb.WriteString(fmt.Sprintf("📚 *%s* — *%s*\n\n", q.Category, q.Level))

    // Question
    sb.WriteString("*Pertanyaan:*\n")
    sb.WriteString(fmt.Sprintf("%s\n\n", escapeMarkdown(q.Question)))

    // Answer
    sb.WriteString("💡 *Jawaban:*\n")
    sb.WriteString(fmt.Sprintf("%s\n\n", escapeMarkdown(q.Answer)))

    // Follow-ups (jika ada)
    if len(q.FollowUps) > 0 {
        sb.WriteString("🔍 *Follow-up:*\n")
        for i, followUp := range q.FollowUps {
            sb.WriteString(fmt.Sprintf("• %s\n", escapeMarkdown(followUp)))
            if i >= 2 { // Max 3 follow-ups
                break
            }
        }
        sb.WriteString("\n")
    }

    // Tags
    if len(q.Tags) > 0 {
        sb.WriteString("---\n")
        sb.WriteString(fmt.Sprintf("_Tags: %s_", strings.Join(q.Tags, ", ")))
    }

    return sb.String()
}
```

### 3.3 Markdown Escaping

```go
// escapeMarkdown escapes special characters untuk Markdown format
func escapeMarkdown(text string) string {
    // Characters yang perlu di-escape: _ * [ ] ( ) ~ ` > # + - = | { } . !
    specialChars := []string{
        "_", "*", "[", "]", "(", ")", "~", "`", 
        ">", "#", "+", "-", "=", "|", "{", "}", ".", "!",
    }
    
    result := text
    for _, char := range specialChars {
        result = strings.ReplaceAll(result, char, "\\"+char)
    }
    return result
}
```

### 3.4 Message Length Limit

Telegram memiliki limit **4096 characters** per message.

```go
const MaxMessageLength = 4096

func TruncateMessage(text string, maxLength int) string {
    if len(text) <= maxLength {
        return text
    }
    
    // Truncate dan tambahkan "..."
    truncated := text[:maxLength-3]
    
    // Cari last newline untuk avoid cut di tengah baris
    if idx := strings.LastIndex(truncated, "\n"); idx > 0 {
        truncated = truncated[:idx]
    }
    
    return truncated + "..."
}
```

---

## 4. Advanced Features

### 4.1 Send Message with Inline Keyboard

```go
func SendQuestionWithKeyboard(chatID int64, q *Question) error {
    msg := tgbotapi.NewMessage(chatID, FormatQuestionMessage(q))
    msg.ParseMode = "Markdown"

    // Create inline keyboard
    keyboard := tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonURL("📖 Learn More", "https://example.com/"+strconv.Itoa(q.ID)),
        ),
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("🔄 Next Question", "next_question"),
            tgbotapi.NewInlineKeyboardButtonData("⭐ Save", "save_"+strconv.Itoa(q.ID)),
        ),
    )

    msg.ReplyMarkup = keyboard
    _, err := bot.Send(msg)
    return err
}
```

### 4.2 Send Photo with Caption

```go
func SendPhotoWithCaption(chatID int64, photoURL, caption string) error {
    msg := tgbotapi.NewPhotoShare(chatID, photoURL)
    msg.Caption = caption
    msg.ParseMode = "Markdown"
    
    _, err := bot.Send(msg)
    return err
}
```

### 4.3 Edit Message (for future interactive features)

```go
func EditMessageText(chatID int64, messageID int, newText string) error {
    edit := tgbotapi.NewEditMessageText(chatID, messageID, newText)
    edit.ParseMode = "Markdown"
    
    _, err := bot.Send(edit)
    return err
}
```

---

## 5. Error Handling

### 5.1 Common Telegram Errors

```go
import (
    "errors"
    "golang.org/x/xerrors"
)

func HandleTelegramError(err error) error {
    if err == nil {
        return nil
    }

    var apiError *tgbotapi.Error
    if xerrors.As(err, &apiError) {
        switch apiError.Code {
        case 400:
            return fmt.Errorf("bad request (invalid parameters): %w", err)
        case 401:
            return fmt.Errorf("unauthorized (invalid bot token): %w", err)
        case 403:
            return fmt.Errorf("forbidden (bot blocked by user): %w", err)
        case 404:
            return fmt.Errorf("not found (chat not found): %w", err)
        case 429:
            return fmt.Errorf("rate limit exceeded: %w", err)
        default:
            return fmt.Errorf("telegram API error %d: %w", apiError.Code, err)
        }
    }

    return err
}
```

### 5.2 Retryable Errors

```go
func IsRetryableError(err error) bool {
    if err == nil {
        return false
    }

    var apiError *tgbotapi.Error
    if xerrors.As(err, &apiError) {
        // Retry on rate limit (429) or server errors (5xx)
        return apiError.Code == 429 || apiError.Code >= 500
    }

    // Retry on network errors
    var netErr net.Error
    if xerrors.As(err, &netErr) && netErr.Timeout() {
        return true
    }

    return false
}
```

### 5.3 Non-Retryable Errors

```go
func IsNonRetryableError(err error) bool {
    if err == nil {
        return false
    }

    var apiError *tgbotapi.Error
    if xerrors.As(err, &apiError) {
        // Don't retry on client errors (4xx, except 429)
        return apiError.Code >= 400 && apiError.Code < 500 && apiError.Code != 429
    }

    return false
}
```

---

## 6. Chat ID Whitelist

### 6.1 Whitelist Implementation

```go
type Whitelist struct {
    allowedIDs map[int64]bool
    logger     *logrus.Entry
}

func NewWhitelist(chatIDs []int64, logger *logrus.Entry) *Whitelist {
    allowed := make(map[int64]bool)
    for _, id := range chatIDs {
        allowed[id] = true
    }

    return &Whitelist{
        allowedIDs: allowed,
        logger:     logger,
    }
}

func (w *Whitelist) IsAllowed(chatID int64) bool {
    return w.allowedIDs[chatID]
}

func (w *Whitelist) Validate(chatID int64) error {
    if !w.IsAllowed(chatID) {
        w.logger.WithField("chat_id", chatID).Warn("Unauthorized access attempt")
        return fmt.Errorf("chat ID %d is not whitelisted", chatID)
    }
    return nil
}
```

### 6.2 Usage in Delivery

```go
func (c *TelegramClient) SendToWhitelisted(chatID int64, text string, whitelist *Whitelist) error {
    // Validate chat ID
    if err := whitelist.Validate(chatID); err != nil {
        return err
    }

    // Send message
    return c.SendMessage(chatID, text)
}
```

---

## 7. Rate Limiting

### 7.1 Telegram Rate Limits

- **Group messages:** 30 messages per second
- **Private chats:** 1 message per second (soft limit)
- **Broadcast:** 1000 recipients per second

**Navisha Spark:** 8 messages/day (tiap 3 jam) — jauh di bawah limit.

### 7.2 Rate Limiter Implementation

```go
type RateLimiter struct {
    lastCall time.Time
    mu       sync.Mutex
}

func NewRateLimiter() *RateLimiter {
    return &RateLimiter{}
}

func (rl *RateLimiter) Wait() {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    elapsed := time.Since(rl.lastCall)
    if elapsed < 1*time.Second {
        sleepTime := 1*time.Second - elapsed
        time.Sleep(sleepTime)
    }
    
    rl.lastCall = time.Now()
}

// Usage
limiter := NewRateLimiter()

func (c *TelegramClient) SendWithRateLimit(chatID int64, text string) error {
    limiter.Wait()
    return c.SendMessage(chatID, text)
}
```

---

## 8. Webhook vs Polling

### 8.1 Why Navisha Spark Uses SendMessage Only

Navisha Spark hanya menggunakan **sendMessage** (outgoing messages), tidak menerima incoming messages. Oleh karena itu:

- **Tidak butuh webhook** (untuk receive updates)
- **Tidak butuh polling** (getUpdates)
- **Tidak butuh long-polling**

### 8.2 If You Need to Receive Messages (Future)

```go
// Webhook approach (untuk future jika perlu interactive bot)
func StartWebhook(bot *tgbotapi.BotAPI, webhookURL string) {
    _, err := bot.SetWebhook(tgbotapi.NewWebhookWithCert(webhookURL, "cert.pem"))
    if err != nil {
        logrus.Fatal("Failed to set webhook:", err)
    }

    updates := bot.ListenForWebhook("/")
    for update := range updates {
        go HandleUpdate(update)
    }
}

// Polling approach (simpler, untuk development)
func StartPolling(bot *tgbotapi.BotAPI) {
    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60
    updates := bot.GetUpdatesChan(u)

    for update := range updates {
        go HandleUpdate(update)
    }
}
```

---

## 9. Message Templates

### 9.1 Question Template

```go
const QuestionTemplate = `📚 *%s* — *%s*

*Pertanyaan:*
%s

💡 *Jawaban:*
%s

%s
---
_Tags: %s_`

func FormatQuestion(q *Question) string {
    followUps := ""
    if len(q.FollowUps) > 0 {
        followUps = "🔍 *Follow-up:*\n"
        for i, fu := range q.FollowUps {
            if i >= 3 {
                break
            }
            followUps += fmt.Sprintf("• %s\n", escapeMarkdown(fu))
        }
    }

    return fmt.Sprintf(QuestionTemplate,
        escapeMarkdown(q.Category),
        escapeMarkdown(q.Level),
        escapeMarkdown(q.Question),
        escapeMarkdown(q.Answer),
        followUps,
        strings.Join(q.Tags, ", "),
    )
}
```

### 9.2 Error Notification Template

```go
const ErrorTemplate = `⚠️ *Navisha Spark — Error Notification*

*Error:* %s
*Time:* %s
*Category:* %s
*Level:* %s

_Skipping to next schedule..._`

func FormatErrorNotification(err error, category, level string) string {
    return fmt.Sprintf(ErrorTemplate,
        escapeMarkdown(err.Error()),
        time.Now().Format("2006-01-02 15:04:05 MST"),
        escapeMarkdown(category),
        escapeMarkdown(level),
    )
}
```

### 9.3 Startup Notification (Optional)

```go
const StartupTemplate = `✅ *Navisha Spark Started*

*Status:* Running
*Schedule:* Every 3 hours
*Timezone:* Asia/Jakarta (WIB)
*Database:* Connected
*Telegram:* Connected

_Ready to send learning insights!_`

func FormatStartupNotification() string {
    return StartupTemplate
}
```

---

## 10. Testing

### 10.1 Unit Test with Mock

```go
import "github.com/stretchr/testify/mock"

type MockTelegramClient struct {
    mock.Mock
}

func (m *MockTelegramClient) Send(ctx context.Context, msg *tgbotapi.MessageConfig) error {
    args := m.Called(ctx, msg)
    return args.Error(0)
}

func TestSendQuestion(t *testing.T) {
    mockClient := new(MockTelegramClient)
    mockClient.On("Send", mock.Anything, mock.Anything).Return(nil)

    client := &TelegramClient{bot: mockClient}
    
    q := &Question{
        Category: "Golang",
        Level:    "intermediate",
        Question: "Test question?",
        Answer:   "Test answer",
    }

    err := client.SendQuestion(203294061, q)
    if err != nil {
        t.Errorf("SendQuestion() error = %v", err)
    }

    mockClient.AssertExpectations(t)
}
```

### 10.2 Integration Test (with Test Bot)

```go
func TestSendMessageToTestBot(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    bot, err := tgbotapi.NewBotAPI(os.Getenv("TEST_BOT_TOKEN"))
    if err != nil {
        t.Fatal(err)
    }

    msg := tgbotapi.NewMessage(203294061, "Test message from Navisha Spark")
    msg.ParseMode = "Markdown"

    _, err = bot.Send(msg)
    if err != nil {
        t.Fatalf("Failed to send test message: %v", err)
    }

    t.Log("Test message sent successfully")
}
```

---

## 11. Common Pitfalls

### 11.1 Markdown Parsing Errors

```go
// ❌ Bad - unescaped special characters
msg.Text = "*Bold* _italic_ `code` [link](url)"

// ✅ Good - escape special characters
msg.Text = fmt.Sprintf("*%s* _%s_ `%s`", 
    escapeMarkdown("Bold"),
    escapeMarkdown("italic"),
    escapeMarkdown("code"),
)
```

### 11.2 Message Too Long

```go
// ❌ Bad - no length check
msg.Text = longText // Might exceed 4096 chars

// ✅ Good - truncate if needed
msg.Text = TruncateMessage(longText, MaxMessageLength)
```

### 11.3 Ignoring Errors

```go
// ❌ Bad - ignoring error
bot.Send(msg)

// ✅ Good - always check error
_, err := bot.Send(msg)
if err != nil {
    return fmt.Errorf("failed to send message: %w", err)
}
```

### 11.4 Not Using Context

```go
// ❌ Bad - no timeout
bot.Send(msg)

// ✅ Good - with timeout
ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
defer cancel()

// Note: tgbotapi doesn't support context directly
// Use channel + select for timeout
done := make(chan bool)
go func() {
    bot.Send(msg)
    done <- true
}()

select {
case <-done:
    // Success
case <-ctx.Done():
    return ctx.Err()
}
```

---

## 12. Best Practices

### 12.1 Do's

✅ **Always escape Markdown special characters**  
✅ **Check message length (max 4096 chars)**  
✅ **Handle errors properly**  
✅ **Use chat ID whitelist**  
✅ **Log all sent messages**  
✅ **Use ParseMode = "Markdown"**  
✅ **Disable web page preview for links**  
✅ **Test with test bot first**

### 12.2 Don'ts

❌ **Don't send messages without escaping**  
❌ **Don't ignore rate limits**  
❌ **Don't hardcode bot token**  
❌ **Don't expose bot to public without whitelist**  
❌ **Don't send messages longer than 4096 chars**  
❌ **Don't use webhook if you only send messages**  
❌ **Don't forget to handle 403 (user blocked bot)**

---

## 13. Monitoring

### 13.1 Log Sent Messages

```go
func (c *TelegramClient) SendWithLogging(chatID int64, text string, logger *logrus.Entry) error {
    logger.WithFields(logrus.Fields{
        "chat_id": chatID,
        "length":  len(text),
    }).Info("Sending message to telegram")

    err := c.SendMessage(chatID, text)
    if err != nil {
        logger.WithError(err).WithField("chat_id", chatID).Error("Failed to send message")
        return err
    }

    logger.Info("Message sent successfully")
    return nil
}
```

### 13.2 Track Message IDs

```go
type DeliveryLog struct {
    MessageID      int
    QuestionID     int
    ChatID         int64
    SentAt         time.Time
    Status         string
    ErrorMessage   string
}

func (c *TelegramClient) SendAndLog(chatID int64, q *Question, logger *logrus.Entry) (*DeliveryLog, error) {
    msg := tgbotapi.NewMessage(chatID, FormatQuestion(q))
    msg.ParseMode = "Markdown"

    sentMsg, err := c.bot.Send(msg)
    if err != nil {
        return nil, err
    }

    log := &DeliveryLog{
        MessageID:  sentMsg.MessageID,
        QuestionID: q.ID,
        ChatID:     chatID,
        SentAt:     time.Now(),
        Status:     "success",
    }

    logger.WithFields(logrus.Fields{
        "message_id": log.MessageID,
        "question_id": log.QuestionID,
    }).Info("Message delivered")

    return log, nil
}
```

---

## 14. Security

### 14.1 Bot Token Security

```go
// ❌ Bad - hardcoded token
bot, _ := tgbotapi.NewBotAPI("123456789:ABCdef...")

// ✅ Good - from environment variable
bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
if err != nil {
    logrus.Fatal("Failed to create bot:", err)
}
```

### 14.2 Chat ID Validation

```go
// Always validate chat ID before sending
func (c *TelegramClient) SendSecure(chatID int64, text string, whitelist *Whitelist) error {
    // Validate chat ID
    if err := whitelist.Validate(chatID); err != nil {
        return err
    }

    // Send message
    return c.SendMessage(chatID, text)
}
```

---

## 15. Troubleshooting

### 15.1 Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| `401 Unauthorized` | Invalid bot token | Check bot token di `.env` |
| `403 Forbidden` | User blocked bot | User harus start bot lagi |
| `400 Bad Request` | Invalid Markdown | Escape special characters |
| `429 Too Many Requests` | Rate limit exceeded | Implement rate limiter |
| `404 Not Found` | Chat ID not found | Verify chat ID dengan getUpdates |

### 15.2 Debug Mode

```go
// Enable debug mode untuk development
bot.Debug = true

// Log all API requests
bot.Request(tgbotapi.NewGetMe())
```

---

## 16. API Reference

### 16.1 Most Used Methods

| Method | Purpose | Example |
|--------|---------|---------|
| `Send` | Send message | `bot.Send(msg)` |
| `NewMessage` | Create text message | `tgbotapi.NewMessage(chatID, text)` |
| `NewPhotoShare` | Send photo | `tgbotapi.NewPhotoShare(chatID, url)` |
| `NewEditMessageText` | Edit message | `tgbotapi.NewEditMessageText(chatID, msgID, text)` |
| `GetMe` | Get bot info | `bot.GetMe()` |
| `GetUpdates` | Get updates (polling) | `bot.GetUpdatesChan(updateConfig)` |
| `SetWebhook` | Set webhook URL | `bot.SetWebhook(config)` |

### 16.2 MessageConfig Fields

```go
msg := tgbotapi.MessageConfig{
    BaseChat: tgbotapi.BaseChat{
        ChatID:           chatID,
        ReplyToMessageID: 0,  // Optional: reply to message
    },
    Text:                  "Message text",
    ParseMode:             "Markdown",  // or "HTML", ""
    DisableWebPagePreview: true,
    DisableNotification:   false,
    ReplyMarkup:           nil,  // Inline keyboard
}
```

---

**Document End**