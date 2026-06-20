package telegram

import (
	"fmt"
	"sync"
)

// Whitelist manages allowed chat IDs for receiving messages.
type Whitelist struct {
	mu    sync.RWMutex
	chats map[int64]bool
}

// NewWhitelist creates a new Whitelist.
func NewWhitelist() *Whitelist {
	return &Whitelist{
		chats: make(map[int64]bool),
	}
}

// NewWhitelistWithIDs creates a new Whitelist with initial chat IDs.
func NewWhitelistWithIDs(chatIDs []int64) *Whitelist {
	w := NewWhitelist()
	for _, id := range chatIDs {
		w.chats[id] = true
	}
	return w
}

// IsAllowed checks if a chat ID is allowed.
func (w *Whitelist) IsAllowed(chatID int64) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.chats[chatID]
}

// IsAllowedString checks if a string representation of chat ID is allowed.
func (w *Whitelist) IsAllowedString(chatID string) bool {
	var id int64
	if _, err := fmt.Sscanf(chatID, "%d", &id); err != nil {
		return false
	}
	return w.IsAllowed(id)
}

// Add adds a chat ID to the whitelist.
func (w *Whitelist) Add(chatID int64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.chats[chatID] = true
}

// AddString adds a string chat ID to the whitelist.
func (w *Whitelist) AddString(chatID string) error {
	var id int64
	if _, err := fmt.Sscanf(chatID, "%d", &id); err != nil {
		return fmt.Errorf("invalid chat ID: %s", chatID)
	}
	w.Add(id)
	return nil
}

// Remove removes a chat ID from the whitelist.
func (w *Whitelist) Remove(chatID int64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.chats, chatID)
}

// GetAll returns all allowed chat IDs.
func (w *Whitelist) GetAll() []int64 {
	w.mu.RLock()
	defer w.mu.RUnlock()

	ids := make([]int64, 0, len(w.chats))
	for id := range w.chats {
		ids = append(ids, id)
	}
	return ids
}

// Count returns the number of allowed chat IDs.
func (w *Whitelist) Count() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.chats)
}

// VerifyAccess verifies that a chat ID is allowed. Returns error if not.
func (w *Whitelist) VerifyAccess(chatID int64) error {
	if !w.IsAllowed(chatID) {
		return fmt.Errorf("chat ID %d is not in the whitelist", chatID)
	}
	return nil
}
