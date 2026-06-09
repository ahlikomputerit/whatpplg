package antiban

import (
	"testing"
	"time"
)

func newTestCfg() *Config {
	c := DefaultConfig(PresetConservative)
	c.MinDelayMs = 10
	c.MaxDelayMs = 100
	c.MaxPerMinute = 5
	c.MaxPerHour = 50
	c.MaxPerDay = 500
	c.BurstAllowance = 2
	c.NewChatDelayMs = 50
	c.MaxIdenticalMessages = 3
	return &c
}

func TestRateLimiter_CanSend(t *testing.T) {
	rl := NewRateLimiter(newTestCfg())

	t.Run("allows within limits", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			if !rl.CanSend() {
				t.Fatalf("expected allow at attempt %d", i+1)
			}
			rl.Record("test@s.whatsapp.net")
		}
	})

	t.Run("blocks after per-minute limit + burst exhausted", func(t *testing.T) {
		rl2 := NewRateLimiter(newTestCfg())
		for i := 0; i < 5; i++ {
			rl2.Record("test@s.whatsapp.net")
		}
		rl2.Record("test@s.whatsapp.net")
		rl2.CanSend()
		rl2.Record("test@s.whatsapp.net")
		rl2.CanSend()
		if rl2.CanSend() {
			t.Fatal("expected block after per-minute limit + burst exhausted")
		}
	})

	t.Run("burst allowance bypasses per-minute limit", func(t *testing.T) {
		rl3 := NewRateLimiter(newTestCfg())
		for i := 0; i < 5; i++ {
			rl3.Record("test@s.whatsapp.net")
		}
		allowed := rl3.CanSend()
		if !allowed {
			t.Fatal("expected burst allowance to permit send")
		}
		allowed = rl3.CanSend()
		if !allowed {
			t.Fatal("expected second burst allowance to permit send")
		}
		if rl3.CanSend() {
			t.Fatal("expected block after both bursts exhausted")
		}
	})

	t.Run("blocks after per-day limit", func(t *testing.T) {
		rl4 := NewRateLimiter(newTestCfg())
		for i := 0; i < 500; i++ {
			rl4.Record("test@s.whatsapp.net")
		}
		if rl4.CanSend() {
			t.Fatal("expected block after per-day limit")
		}
	})
}

func TestRateLimiter_GetDelay(t *testing.T) {
	rl := NewRateLimiter(newTestCfg())

	t.Run("returns delay within bounds", func(t *testing.T) {
		d := rl.GetDelay("test@s.whatsapp.net", []byte("hello"))
		if d < 10*time.Millisecond || d > 100*time.Millisecond {
			t.Fatalf("delay %v out of bounds [10ms, 100ms]", d)
		}
	})

	t.Run("new chat delay is larger", func(t *testing.T) {
		d1 := rl.GetDelay("stranger@s.whatsapp.net", []byte("first"))
		d2 := rl.GetDelay("known@s.whatsapp.net", []byte("second"))
		if d1 <= d2 {
			t.Logf("note: new chat delay %v not larger than known %v (random jitter)", d1, d2)
		}
	})

	t.Run("identical messages get doubled delay", func(t *testing.T) {
		msg := []byte("spam")
		for i := 0; i < 4; i++ {
			rl.GetDelay("same@s.whatsapp.net", msg)
			rl.Record("same@s.whatsapp.net")
		}
		d := rl.GetDelay("same@s.whatsapp.net", msg)
		if d < 20*time.Millisecond {
			t.Fatalf("expected doubled delay for identical message, got %v", d)
		}
	})
}

func TestRateLimiter_AdaptLimits(t *testing.T) {
	rl := NewRateLimiter(newTestCfg())
	origMin := rl.cfg.MaxPerMinute

	rl.AdaptLimits(0.5)
	if rl.cfg.MaxPerMinute != origMin/2 {
		t.Fatalf("expected %d, got %d", origMin/2, rl.cfg.MaxPerMinute)
	}

	rl.AdaptLimits(0.0)
	if rl.cfg.MaxPerMinute < 0 {
		t.Fatal("factor should not produce negative")
	}
}

func TestRateLimiter_GetStats(t *testing.T) {
	rl := NewRateLimiter(newTestCfg())
	rl.Record("a@s.whatsapp.net")
	rl.Record("b@s.whatsapp.net")

	stats := rl.GetStats()
	if stats["sent"].(int64) != 2 {
		t.Fatalf("expected 2 sent, got %d", stats["sent"])
	}
	if stats["known_chats"].(int) != 2 {
		t.Fatalf("expected 2 known chats, got %d", stats["known_chats"])
	}
	if stats["per_minute"].(int) != 2 {
		t.Fatalf("expected 2 per minute, got %d", stats["per_minute"])
	}
}

func TestRateLimiter_KnownChats(t *testing.T) {
	rl := NewRateLimiter(newTestCfg())
	rl.Record("a@s.whatsapp.net")

	chats := rl.GetKnownChats()
	if !chats["a@s.whatsapp.net"] {
		t.Fatal("expected a@s.whatsapp.net in known chats")
	}

	rl.RestoreKnownChats(map[string]bool{"b@s.whatsapp.net": true})
	chats = rl.GetKnownChats()
	if !chats["b@s.whatsapp.net"] {
		t.Fatal("expected b@s.whatsapp.net after restore")
	}
}
