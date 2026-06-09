package antiban

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	t.Run("conservative preset", func(t *testing.T) {
		cfg := DefaultConfig(PresetConservative)
		if cfg.MaxPerMinute != 2 {
			t.Fatalf("expected MaxPerMinute 2, got %d", cfg.MaxPerMinute)
		}
		if cfg.MaxPerHour != 30 {
			t.Fatalf("expected MaxPerHour 30, got %d", cfg.MaxPerHour)
		}
		if cfg.MaxPerDay != 150 {
			t.Fatalf("expected MaxPerDay 150, got %d", cfg.MaxPerDay)
		}
		if cfg.CircuitBreakerCooldown == 0 {
			t.Fatal("expected non-zero CircuitBreakerCooldown")
		}
	})

	t.Run("moderate preset", func(t *testing.T) {
		cfg := DefaultConfig(PresetModerate)
		if cfg.MaxPerMinute != 5 {
			t.Fatalf("expected MaxPerMinute 5, got %d", cfg.MaxPerMinute)
		}
		if cfg.MaxPerHour != 60 {
			t.Fatalf("expected MaxPerHour 60, got %d", cfg.MaxPerHour)
		}
	})

	t.Run("aggressive preset", func(t *testing.T) {
		cfg := DefaultConfig(PresetAggressive)
		if cfg.MaxPerMinute != 12 {
			t.Fatalf("expected MaxPerMinute 12, got %d", cfg.MaxPerMinute)
		}
	})

	t.Run("high-volume preset", func(t *testing.T) {
		cfg := DefaultConfig(PresetHighVolume)
		if cfg.MaxPerMinute != 30 {
			t.Fatalf("expected MaxPerMinute 30, got %d", cfg.MaxPerMinute)
		}
		if cfg.MaxPerDay != 2000 {
			t.Fatalf("expected MaxPerDay 2000, got %d", cfg.MaxPerDay)
		}
	})
}

func TestResolveConfig(t *testing.T) {
	t.Run("preset only", func(t *testing.T) {
		cfg := ResolveConfig(PresetConservative, Config{})
		if cfg.MaxPerMinute != 2 {
			t.Fatalf("expected MaxPerMinute 2, got %d", cfg.MaxPerMinute)
		}
	})

	t.Run("override max per minute", func(t *testing.T) {
		cfg := ResolveConfig(PresetConservative, Config{MaxPerMinute: 10})
		if cfg.MaxPerMinute != 10 {
			t.Fatalf("expected MaxPerMinute 10, got %d", cfg.MaxPerMinute)
		}
		if cfg.MaxPerHour != 30 {
			t.Fatalf("expected MaxPerHour to come from preset (30), got %d", cfg.MaxPerHour)
		}
	})

	t.Run("override warmup days", func(t *testing.T) {
		cfg := ResolveConfig(PresetModerate, Config{WarmUpDays: 20})
		if cfg.WarmUpDays != 20 {
			t.Fatalf("expected WarmUpDays 20, got %d", cfg.WarmUpDays)
		}
	})
}

func TestConfig_ApplyPreset(t *testing.T) {
	cfg := DefaultConfig(PresetConservative)
	cfg.applyPreset()
	if cfg.MaxPerMinute != 2 {
		t.Fatalf("expected MaxPerMinute 2 after applyPreset, got %d", cfg.MaxPerMinute)
	}
}

func TestDefaultConfig_Defaults(t *testing.T) {
	cfg := DefaultConfig(PresetConservative)
	if cfg.ActiveHourStart != 8 {
		t.Fatalf("expected ActiveHourStart 8, got %d", cfg.ActiveHourStart)
	}
	if cfg.ActiveHourEnd != 22 {
		t.Fatalf("expected ActiveHourEnd 22, got %d", cfg.ActiveHourEnd)
	}
	if cfg.WeekendFactor != 0.6 {
		t.Fatalf("expected WeekendFactor 0.6, got %f", cfg.WeekendFactor)
	}
	if cfg.TimelockBlockDuration == 0 {
		t.Fatal("expected non-zero TimelockBlockDuration")
	}
	if cfg.EnableTypoInjection {
		t.Fatal("expected EnableTypoInjection false by default")
	}
}
