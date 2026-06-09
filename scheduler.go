package antiban

import (
	"sync"
	"time"
)

type Scheduler struct {
	mu sync.Mutex
	cfg *Config
}

func NewScheduler(cfg *Config) *Scheduler {
	return &Scheduler{cfg: cfg}
}

func (s *Scheduler) IsActiveTime() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isActiveTime(time.Now())
}

func (s *Scheduler) isActiveTime(now time.Time) bool {
	hour := now.Hour()
	weekday := now.Weekday()

	if s.cfg.ActiveHourStart <= s.cfg.ActiveHourEnd {
		if hour < s.cfg.ActiveHourStart || hour >= s.cfg.ActiveHourEnd {
			return false
		}
	} else {
		if hour < s.cfg.ActiveHourStart && hour >= s.cfg.ActiveHourEnd {
			return false
		}
	}

	if weekday == time.Saturday || weekday == time.Sunday {
		return true
	}

	return true
}

func (s *Scheduler) GetSpeedFactor() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	weekday := now.Weekday()
	hour := now.Hour()

	factor := 1.0

	if weekday == time.Saturday || weekday == time.Sunday {
		factor *= s.cfg.WeekendFactor
	}

	if hour >= s.cfg.PeakHourStart && hour < s.cfg.PeakHourEnd {
		factor *= (1 + s.cfg.PeakBoost)
	}

	if hour >= 12 && hour <= 14 {
		factor *= 0.7
	}

	return factor
}

func (s *Scheduler) AdjustDelay(delay time.Duration) time.Duration {
	factor := s.GetSpeedFactor()
	return time.Duration(float64(delay) / factor)
}

func (s *Scheduler) MsUntilActive() time.Duration {
	if s.IsActiveTime() {
		return 0
	}

	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), s.cfg.ActiveHourStart, 0, 0, 0, now.Location())

	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}

	return time.Until(next)
}

func (s *Scheduler) GetStatus() map[string]any {
	return map[string]any{
		"active":    s.IsActiveTime(),
		"speed":     s.GetSpeedFactor(),
		"next_ms":   s.MsUntilActive().Milliseconds(),
	}
}
