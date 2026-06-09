package antiban

import (
	"testing"
	"time"
)

func TestJidCircuitBreaker_New(t *testing.T) {
	jcb := NewJidCircuitBreaker(newTestCfg())
	if jcb == nil {
		t.Fatal("expected non-nil JidCircuitBreaker")
	}
}

func TestJidCircuitBreaker_CanSend_UnknownJID(t *testing.T) {
	jcb := NewJidCircuitBreaker(newTestCfg())
	if !jcb.CanSend("unknown@s.whatsapp.net") {
		t.Fatal("expected allow for unknown JID")
	}
}

func TestJidCircuitBreaker_RecordFailure_Opens(t *testing.T) {
	cfg := newTestCfg()
	cfg.CircuitBreakerThreshold = 2
	jcb := NewJidCircuitBreaker(cfg)

	jcb.RecordFailure("bad@s.whatsapp.net")
	if !jcb.CanSend("bad@s.whatsapp.net") {
		t.Fatal("expected still closed after 1 failure")
	}

	jcb.RecordFailure("bad@s.whatsapp.net")
	if jcb.CanSend("bad@s.whatsapp.net") {
		t.Fatal("expected open after 2 failures")
	}
}

func TestJidCircuitBreaker_RecordSuccess_Closes(t *testing.T) {
	cfg := newTestCfg()
	cfg.CircuitBreakerThreshold = 1
	jcb := NewJidCircuitBreaker(cfg)

	jcb.RecordFailure("jid@s.whatsapp.net")
	jcb.RecordSuccess("jid@s.whatsapp.net")

	if !jcb.CanSend("jid@s.whatsapp.net") {
		t.Fatal("expected closed after success")
	}
}

func TestJidCircuitBreaker_HalfOpenAfterCooldown(t *testing.T) {
	cfg := newTestCfg()
	cfg.CircuitBreakerThreshold = 1
	cfg.CircuitBreakerCooldown = 1 * time.Millisecond
	jcb := NewJidCircuitBreaker(cfg)

	jcb.RecordFailure("jid@s.whatsapp.net")
	time.Sleep(5 * time.Millisecond)

	if !jcb.CanSend("jid@s.whatsapp.net") {
		t.Fatal("expected half-open after cooldown")
	}
}

func TestJidCircuitBreaker_GetJitter(t *testing.T) {
	cfg := newTestCfg()
	cfg.CircuitBreakerThreshold = 1
	jcb := NewJidCircuitBreaker(cfg)

	jitter := jcb.GetJitter("unknown@s.whatsapp.net")
	if jitter != 0 {
		t.Fatal("expected 0 jitter for non-blocked JID")
	}

	jcb.RecordFailure("bad@s.whatsapp.net")
	jitter = jcb.GetJitter("bad@s.whatsapp.net")
	if jitter <= 0 {
		t.Fatal("expected positive jitter for blocked JID")
	}
}

func TestJidCircuitBreaker_GetStats(t *testing.T) {
	cfg := newTestCfg()
	cfg.CircuitBreakerThreshold = 1
	jcb := NewJidCircuitBreaker(cfg)
	jcb.RecordFailure("bad@s.whatsapp.net")

	stats := jcb.GetStats()
	if stats["total"].(int) != 1 {
		t.Fatalf("expected 1 total, got %d", stats["total"])
	}
	if stats["open"].(int) != 1 {
		t.Fatalf("expected 1 open, got %d", stats["open"])
	}
}

func TestJidCircuitBreaker_EvictStale(t *testing.T) {
	jcb := NewJidCircuitBreaker(newTestCfg())
	jcb.RecordFailure("old@s.whatsapp.net")
	jcb.RecordSuccess("old@s.whatsapp.net")
	stats := jcb.GetStats()
	_ = stats
}
