package antiban

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStateManager_New(t *testing.T) {
	sm := NewStateManager(newTestCfg())
	if sm == nil {
		t.Fatal("expected non-nil StateManager")
	}
}

func TestStateManager_SaveAndGetState(t *testing.T) {
	sm := NewStateManager(newTestCfg())
	sm.SaveWarmupState(3, 15, 1.8, testTime)
	sm.SaveKnownChats(map[string]bool{"user1@s.whatsapp.net": true})

	state := sm.GetPersistedState()
	if state.WarmupDay != 3 {
		t.Fatalf("expected WarmupDay 3, got %d", state.WarmupDay)
	}
	if state.WarmupDailyCount != 15 {
		t.Fatalf("expected WarmupDailyCount 15, got %d", state.WarmupDailyCount)
	}
	if state.KnownChats["user1@s.whatsapp.net"] != true {
		t.Fatal("expected user1 in known chats")
	}
}

func TestStateManager_FlushAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	sm1 := NewStateManager(newTestCfg())
	sm1.SetPath(path)
	sm1.SaveWarmupState(5, 20, 2.0, testTime)
	sm1.SaveKnownChats(map[string]bool{"user1@s.whatsapp.net": true})
	err := sm1.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	sm2 := NewStateManager(newTestCfg())
	sm2.SetPath(path)
	loaded, err := sm2.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.WarmupDay != 5 {
		t.Fatalf("expected WarmupDay 5, got %d", loaded.WarmupDay)
	}
	if loaded.WarmupDailyCount != 20 {
		t.Fatalf("expected WarmupDailyCount 20, got %d", loaded.WarmupDailyCount)
	}
	if loaded.KnownChats["user1@s.whatsapp.net"] != true {
		t.Fatal("expected user1 in loaded known chats")
	}
}

func TestStateManager_Load_NonExistent(t *testing.T) {
	sm := NewStateManager(newTestCfg())
	sm.SetPath("/nonexistent/path/state.json")
	loaded, err := sm.Load()
	if err != nil {
		t.Fatalf("Load should not error on non-existent file: %v", err)
	}
	if loaded != nil {
		t.Fatal("expected nil for non-existent file")
	}
}

func TestStateManager_Load_CorruptedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.json")
	os.WriteFile(path, []byte("{invalid json"), 0644)

	sm := NewStateManager(newTestCfg())
	sm.SetPath(path)
	_, err := sm.Load()
	if err == nil {
		t.Fatal("expected error for corrupted file")
	}
}

func TestStateManager_Reset(t *testing.T) {
	sm := NewStateManager(newTestCfg())
	sm.SaveWarmupState(3, 15, 1.8, testTime)
	sm.Reset()
	state := sm.GetPersistedState()
	if state.WarmupDay != 0 {
		t.Fatalf("expected WarmupDay 0 after reset, got %d", state.WarmupDay)
	}
	if state.WarmupDailyCount != 0 {
		t.Fatalf("expected WarmupDailyCount 0 after reset, got %d", state.WarmupDailyCount)
	}
}

func TestStateManager_AutoSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "autosave.json")

	sm := NewStateManager(newTestCfg())
	sm.SetPath(path)
	sm.StartAutoSave(50 * time.Millisecond)
	sm.SaveWarmupState(1, 5, 1.5, testTime)
	time.Sleep(100 * time.Millisecond)
	sm.Stop()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected autosaved file: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty autosaved file")
	}
}

func TestStateManager_Stop(t *testing.T) {
	sm := NewStateManager(newTestCfg())
	sm.StartAutoSave(50 * time.Millisecond)
	sm.Stop()
}

func TestStateManager_Flush_NoPath(t *testing.T) {
	sm := NewStateManager(newTestCfg())
	err := sm.Flush()
	if err != nil {
		t.Fatalf("Flush with no path should not error: %v", err)
	}
}
