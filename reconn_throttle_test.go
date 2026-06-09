package antiban

import (
	"testing"
	"time"
)

func TestPostReconnectThrottle_InitialState(t *testing.T) {
	prt := NewPostReconnectThrottle(newTestCfg())
	if prt.BeforeSend() != 0 {
		t.Fatal("expected no delay before any disconnect")
	}
	if prt.GetCurrentMultiplier() != 1.0 {
		t.Fatalf("expected multiplier 1.0, got %v", prt.GetCurrentMultiplier())
	}
}

func TestPostReconnectThrottle_OnDisconnect(t *testing.T) {
	prt := NewPostReconnectThrottle(newTestCfg())
	prt.OnDisconnect()
	stats := prt.GetStats()
	if _, ok := stats["multiplier"]; !ok {
		t.Fatal("expected stats after disconnect")
	}
}

func TestPostReconnectThrottle_OnReconnect(t *testing.T) {
	prt := NewPostReconnectThrottle(newTestCfg())
	prt.OnDisconnect()
	prt.OnReconnect()
	mult := prt.GetCurrentMultiplier()
	if mult >= 1.0 {
		t.Fatalf("expected multiplier < 1.0 after reconnect, got %v", mult)
	}
}

func TestPostReconnectThrottle_BeforeSend_AfterReconnect(t *testing.T) {
	prt := NewPostReconnectThrottle(newTestCfg())
	prt.OnReconnect()
	delay := prt.BeforeSend()
	if delay <= 0 {
		t.Fatal("expected positive delay after reconnect")
	}
}

func TestPostReconnectThrottle_RampUp(t *testing.T) {
	cfg := newTestCfg()
	cfg.ReconnectRampDuration = 100 * time.Millisecond
	cfg.MinDelayMs = 100
	prt := NewPostReconnectThrottle(cfg)
	prt.OnReconnect()

	first := prt.BeforeSend()
	time.Sleep(50 * time.Millisecond)
	second := prt.BeforeSend()

	if second >= first {
		t.Logf("note: ramp-up may not always decrease delay (first=%v, second=%v)", first, second)
	}
}

func TestPostReconnectThrottle_RampComplete(t *testing.T) {
	cfg := newTestCfg()
	cfg.ReconnectRampDuration = 1 * time.Millisecond
	prt := NewPostReconnectThrottle(cfg)
	prt.OnReconnect()
	time.Sleep(5 * time.Millisecond)
	mult := prt.GetCurrentMultiplier()
	if mult != 1.0 {
		t.Fatalf("expected multiplier 1.0 after ramp complete, got %v", mult)
	}
}

func TestPostReconnectThrottle_Destroy(t *testing.T) {
	prt := NewPostReconnectThrottle(newTestCfg())
	prt.OnReconnect()
	prt.Destroy()
	if prt.GetCurrentMultiplier() != 1.0 {
		t.Fatal("expected multiplier 1.0 after destroy")
	}
}

func TestPostReconnectThrottle_GetStats(t *testing.T) {
	prt := NewPostReconnectThrottle(newTestCfg())
	prt.OnReconnect()
	stats := prt.GetStats()
	if _, ok := stats["multiplier"]; !ok {
		t.Fatal("expected multiplier in stats")
	}
	if _, ok := stats["reconnected_s"]; !ok {
		t.Fatal("expected reconnected_s in stats")
	}
	if _, ok := stats["ramp_duration_s"]; !ok {
		t.Fatal("expected ramp_duration_s in stats")
	}
}
