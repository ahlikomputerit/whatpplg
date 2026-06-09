package antiban

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type PersistedState struct {
	WarmupDay         int              `json:"warmup_day"`
	WarmupDailyCount  int              `json:"warmup_daily_count"`
	WarmupGrowth      float64          `json:"warmup_growth"`
	WarmupStartedAt   time.Time        `json:"warmup_started_at"`
	KnownChats        map[string]bool  `json:"known_chats"`
	LastUpdated       time.Time        `json:"last_updated"`
}

type StateManager struct {
	mu         sync.Mutex
	cfg        *Config
	path       string
	state      PersistedState
	dirty      bool
	saveTicker *time.Ticker
	stopCh     chan struct{}
}

func NewStateManager(cfg *Config) *StateManager {
	return &StateManager{
		cfg:    cfg,
		state:  PersistedState{KnownChats: make(map[string]bool)},
		stopCh: make(chan struct{}),
	}
}

func (sm *StateManager) SetPath(path string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.path = path
}

func (sm *StateManager) StartAutoSave(interval time.Duration) {
	sm.mu.Lock()
	sm.saveTicker = time.NewTicker(interval)
	sm.mu.Unlock()

	go func() {
		for {
			select {
			case <-sm.saveTicker.C:
				sm.Flush()
			case <-sm.stopCh:
				return
			}
		}
	}()
}

func (sm *StateManager) Stop() {
	if sm.saveTicker != nil {
		sm.saveTicker.Stop()
	}
	close(sm.stopCh)
	sm.Flush()
}

func (sm *StateManager) SaveWarmupState(warmupDay, dailyCount int, growth float64, startedAt time.Time) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.state.WarmupDay = warmupDay
	sm.state.WarmupDailyCount = dailyCount
	sm.state.WarmupGrowth = growth
	sm.state.WarmupStartedAt = startedAt
	sm.state.LastUpdated = time.Now()
	sm.dirty = true
}

func (sm *StateManager) SaveKnownChats(chats map[string]bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.state.KnownChats = chats
	sm.state.LastUpdated = time.Now()
	sm.dirty = true
}

func (sm *StateManager) GetPersistedState() PersistedState {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.state
}

func (sm *StateManager) Flush() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.dirty || sm.path == "" {
		return nil
	}

	data, err := json.MarshalIndent(sm.state, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(sm.path)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	tmpPath := sm.path + ".tmp"
	err = os.WriteFile(tmpPath, data, 0644)
	if err != nil {
		return err
	}

	err = os.Rename(tmpPath, sm.path)
	if err != nil {
		os.Remove(tmpPath)
		return err
	}

	sm.dirty = false
	return nil
}

func (sm *StateManager) Load() (*PersistedState, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(sm.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var state PersistedState
	err = json.Unmarshal(data, &state)
	if err != nil {
		return nil, err
	}

	sm.state = state
	return &state, nil
}

func (sm *StateManager) Reset() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.state = PersistedState{KnownChats: make(map[string]bool)}
	sm.dirty = true
}
