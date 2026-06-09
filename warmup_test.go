package antiban

import (
	"testing"
	"time"
)

func TestWarmUp_InitialState(t *testing.T) {
	w := NewWarmUp(newTestCfg())
	if w.GetDailyLimit() <= 0 {
		t.Fatal("expected positive daily limit")
	}
	st := w.GetStatus()
	if st["day"].(int) != 1 {
		t.Fatalf("expected day 1, got %d", st["day"])
	}
	if st["daily_count"].(int) != 0 {
		t.Fatalf("expected daily count 0, got %d", st["daily_count"])
	}
}

func TestWarmUp_CanSend(t *testing.T) {
	w := NewWarmUp(newTestCfg())
	if !w.CanSend() {
		t.Fatal("expected CanSend initially")
	}
}

func TestWarmUp_Record(t *testing.T) {
	w := NewWarmUp(newTestCfg())
	w.Record()
	w.Record()
	st := w.GetStatus()
	if st["daily_count"].(int) != 2 {
		t.Fatalf("expected daily count 2, got %d", st["daily_count"])
	}
}

func TestWarmUp_DailyLimitBlocks(t *testing.T) {
	cfg := newTestCfg()
	cfg.WarmUpDays = 1
	cfg.InitialDailyLimit = 3
	cfg.MaxPerDay = 10
	w := NewWarmUp(cfg)
	for w.CanSend() {
		w.Record()
	}
	st := w.GetStatus()
	if st["daily_count"].(int) < 3 {
		t.Fatalf("expected at least 3 sends blocked, got %d", st["daily_count"])
	}
}

func TestWarmUp_ExportImportState(t *testing.T) {
	w := NewWarmUp(newTestCfg())
	w.Record()
	w.Record()

	state := w.ExportState()
	w2 := NewWarmUp(newTestCfg())
	w2.ImportState(state)

	st := w2.GetStatus()
	if st["daily_count"].(int) != 2 {
		t.Fatalf("expected daily count 2 after import, got %d", st["daily_count"])
	}
}

func TestWarmUp_ImportState(t *testing.T) {
	w := NewWarmUp(newTestCfg())
	state := map[string]any{
		"day":           5,
		"daily_count":   10,
		"growth_factor": 2.0,
		"started_at":    time.Now().Add(-48 * time.Hour),
	}
	w.ImportState(state)
	st := w.GetStatus()
	if st["day"].(int) != 5 {
		t.Fatalf("expected day 5, got %d", st["day"])
	}
	if st["daily_count"].(int) != 10 {
		t.Fatalf("expected daily count 10, got %d", st["daily_count"])
	}
}

func TestWarmUp_Reset(t *testing.T) {
	w := NewWarmUp(newTestCfg())
	w.Record()
	w.Record()
	w.Reset()
	st := w.GetStatus()
	if st["day"].(int) != 1 {
		t.Fatalf("expected day 1 after reset, got %d", st["day"])
	}
	if st["daily_count"].(int) != 0 {
		t.Fatalf("expected daily count 0 after reset, got %d", st["daily_count"])
	}
}

func TestWarmUp_GetDailyLimit(t *testing.T) {
	w := NewWarmUp(newTestCfg())
	limit := w.GetDailyLimit()
	if limit <= 0 {
		t.Fatal("expected positive daily limit")
	}
}
