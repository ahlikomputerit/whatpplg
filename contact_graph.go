package antiban

import (
	"sync"
	"time"
)

// ContactInfo tracks the state and messaging activity for a single contact.
type ContactInfo struct {
	State      ContactState
	LastMsgAt  time.Time
	JoinedAt   time.Time
	DailyCount int
	LastDay    int
}

// ContactGraphWarmer manages contact relationships and ensures gradual introduction
// to new contacts to avoid triggering anti-spam detection.
type ContactGraphWarmer struct {
	mu sync.Mutex

	cfg    *Config
	contacts map[string]*ContactInfo
	groups   map[string]time.Time
}

// NewContactGraphWarmer creates a new contact graph warmer.
func NewContactGraphWarmer(cfg *Config) *ContactGraphWarmer {
	return &ContactGraphWarmer{
		cfg:      cfg,
		contacts: make(map[string]*ContactInfo),
		groups:   make(map[string]time.Time),
	}
}

// CanMessage checks whether messaging the given contact is allowed based on relationship state.
func (cgw *ContactGraphWarmer) CanMessage(contactID string, isGroup bool) bool {
	cgw.mu.Lock()
	defer cgw.mu.Unlock()

	if isGroup {
		joinTime, ok := cgw.groups[contactID]
		if ok && cgw.cfg.GroupLurkPeriod > 0 {
			if time.Since(joinTime) < cgw.cfg.GroupLurkPeriod {
				return false
			}
		}
		return true
	}

	contact, exists := cgw.contacts[contactID]
	if !exists {
		return true
	}

	now := time.Now()
	today := now.YearDay()

	if contact.LastDay != today {
		contact.DailyCount = 0
		contact.LastDay = today
	}

	if contact.State == ContactStranger && contact.DailyCount >= cgw.cfg.MaxStrangerPerDay {
		return false
	}

	return true
}

// MarkHandshakeSent transitions a contact from stranger to handshake-sent state.
func (cgw *ContactGraphWarmer) MarkHandshakeSent(contactID string) {
	cgw.mu.Lock()
	defer cgw.mu.Unlock()

	contact, ok := cgw.contacts[contactID]
	if !ok {
		contact = &ContactInfo{State: ContactStranger}
		cgw.contacts[contactID] = contact
	}

	if contact.State == ContactStranger {
		contact.State = ContactHandshakeSent
	}
	contact.LastMsgAt = time.Now()
}

// MarkHandshakeComplete transitions a contact to handshake-complete state.
func (cgw *ContactGraphWarmer) MarkHandshakeComplete(contactID string) {
	cgw.mu.Lock()
	defer cgw.mu.Unlock()

	contact, ok := cgw.contacts[contactID]
	if !ok {
		cgw.contacts[contactID] = &ContactInfo{State: ContactHandshakeComplete, LastMsgAt: time.Now()}
		return
	}

	if contact.State != ContactKnown {
		contact.State = ContactHandshakeComplete
	}
	contact.LastMsgAt = time.Now()
}

// RegisterKnownContact marks a contact as known (established two-way relationship).
func (cgw *ContactGraphWarmer) RegisterKnownContact(contactID string) {
	cgw.mu.Lock()
	defer cgw.mu.Unlock()

	contact, ok := cgw.contacts[contactID]
	if !ok {
		cgw.contacts[contactID] = &ContactInfo{State: ContactKnown, LastMsgAt: time.Now()}
		return
	}
	contact.State = ContactKnown
}

// RegisterGroupJoin records a group join so the lurk period can be enforced.
func (cgw *ContactGraphWarmer) RegisterGroupJoin(groupID string) {
	cgw.mu.Lock()
	defer cgw.mu.Unlock()
	cgw.groups[groupID] = time.Now()
}

// OnIncomingMessage upgrades a sender to known status upon receiving a message from them.
func (cgw *ContactGraphWarmer) OnIncomingMessage(senderID string) {
	cgw.mu.Lock()
	defer cgw.mu.Unlock()

	contact, ok := cgw.contacts[senderID]
	if !ok {
		cgw.contacts[senderID] = &ContactInfo{State: ContactKnown, LastMsgAt: time.Now()}
		return
	}

	contact.State = ContactKnown
	contact.LastMsgAt = time.Now()
}

// GetContactState returns the current relationship state for the given contact.
func (cgw *ContactGraphWarmer) GetContactState(contactID string) ContactState {
	cgw.mu.Lock()
	defer cgw.mu.Unlock()

	contact, ok := cgw.contacts[contactID]
	if !ok {
		return ContactStranger
	}
	return contact.State
}

// RecordMessage increments the daily message count for a contact.
func (cgw *ContactGraphWarmer) RecordMessage(contactID string) {
	cgw.mu.Lock()
	defer cgw.mu.Unlock()

	contact, ok := cgw.contacts[contactID]
	if !ok {
		return
	}

	now := time.Now()
	today := now.YearDay()
	if contact.LastDay != today {
		contact.DailyCount = 0
		contact.LastDay = today
	}
	contact.DailyCount++
	contact.LastMsgAt = now
}
