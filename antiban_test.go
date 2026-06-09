package antiban

import (
	"errors"
	"testing"
	"time"
)

func TestAntiBan_New(t *testing.T) {
	ab := New(PresetConservative, DefaultConfig(PresetConservative))
	if ab == nil {
		t.Fatal("expected non-nil AntiBan")
	}
	if ab.IsPaused() {
		t.Fatal("expected not paused initially")
	}
}

func TestAntiBan_BeforeSend_Allows(t *testing.T) {
	cfg := DefaultConfig(PresetConservative)
	cfg.MinDelayMs = 1
	cfg.MaxDelayMs = 10
	ab := New(PresetConservative, cfg)
	delay, allowed := ab.BeforeSend("test@s.whatsapp.net", []byte("hello"))
	if !allowed {
		t.Fatal("expected allowed")
	}
	if delay <= 0 {
		t.Fatal("expected positive delay")
	}
}

func TestAntiBan_BeforeSend_Paused(t *testing.T) {
	ab := New(PresetConservative, DefaultConfig(PresetConservative))
	ab.Pause()
	_, allowed := ab.BeforeSend("test@s.whatsapp.net", []byte("hello"))
	if allowed {
		t.Fatal("expected blocked when paused")
	}
}

func TestAntiBan_PauseResume(t *testing.T) {
	ab := New(PresetConservative, DefaultConfig(PresetConservative))
	if ab.IsPaused() {
		t.Fatal("expected not paused initially")
	}
	ab.Pause()
	if !ab.IsPaused() {
		t.Fatal("expected paused")
	}
	ab.Resume()
	if ab.IsPaused() {
		t.Fatal("expected not paused after resume")
	}
}

func TestAntiBan_AfterSend(t *testing.T) {
	cfg := DefaultConfig(PresetConservative)
	cfg.MinDelayMs = 1
	cfg.MaxDelayMs = 10
	ab := New(PresetConservative, cfg)
	ab.AfterSend("test@s.whatsapp.net", true)
}

func TestAntiBan_AfterSendFailed(t *testing.T) {
	ab := New(PresetConservative, DefaultConfig(PresetConservative))
	ab.AfterSendFailed("test@s.whatsapp.net", errors.New("send error"))
}

func TestAntiBan_OnDisconnect(t *testing.T) {
	ab := New(PresetConservative, DefaultConfig(PresetConservative))
	ab.OnDisconnect()
}

func TestAntiBan_OnReconnect(t *testing.T) {
	ab := New(PresetConservative, DefaultConfig(PresetConservative))
	ab.OnReconnect()
}

func TestAntiBan_OnIncomingMessage(t *testing.T) {
	ab := New(PresetConservative, DefaultConfig(PresetConservative))
	ab.OnIncomingMessage("sender@s.whatsapp.net")
}

func TestAntiBan_OnDeliveryReceipt(t *testing.T) {
	ab := New(PresetConservative, DefaultConfig(PresetConservative))
	ab.OnDeliveryReceipt()
}

func TestAntiBan_OnStreamError(t *testing.T) {
	ab := New(PresetConservative, DefaultConfig(PresetConservative))
	ab.OnStreamError(errors.New("463 reachout error"))
}

func TestAntiBan_GetStats(t *testing.T) {
	ab := New(PresetConservative, DefaultConfig(PresetConservative))
	stats := ab.GetStats()
	if stats["preset"].(Preset) != PresetConservative {
		t.Fatalf("expected conservative preset, got %v", stats["preset"])
	}
	if _, ok := stats["paused"]; !ok {
		t.Fatal("expected paused field in stats")
	}
	if _, ok := stats["rate_limiter"]; !ok {
		t.Fatal("expected rate_limiter in stats")
	}
	if _, ok := stats["warmup"]; !ok {
		t.Fatal("expected warmup in stats")
	}
	if _, ok := stats["health"]; !ok {
		t.Fatal("expected health in stats")
	}
	if _, ok := stats["scheduler"]; !ok {
		t.Fatal("expected scheduler in stats")
	}
}

func TestAntiBan_Destroy(t *testing.T) {
	ab := New(PresetConservative, DefaultConfig(PresetConservative))
	ab.Destroy()
	_, allowed := ab.BeforeSend("test@s.whatsapp.net", []byte("hello"))
	if allowed {
		t.Fatal("expected blocked after destroy")
	}
}

func TestAntiBan_HealthTriggersPause(t *testing.T) {
	cfg := DefaultConfig(PresetConservative)
	cfg.AutoPauseRiskLevel = 30
	ab := New(PresetConservative, cfg)
	ab.Health.RecordLoggedOut()
	time.Sleep(10 * time.Millisecond)
	if !ab.IsPaused() {
		t.Fatal("expected pause after health critical")
	}
}

func TestAntiBan_BeforeSend_Group(t *testing.T) {
	cfg := DefaultConfig(PresetConservative)
	cfg.MinDelayMs = 1
	cfg.MaxDelayMs = 10
	cfg.GroupLurkPeriod = 0
	ab := New(PresetConservative, cfg)
	delay, allowed := ab.BeforeSend("group@g.us", []byte("hello group"))
	if !allowed {
		t.Fatal("expected allowed for group")
	}
	if delay <= 0 {
		t.Fatal("expected positive delay")
	}
}
