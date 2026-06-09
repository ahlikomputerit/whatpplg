package antiban

import (
	"math/rand/v2"
	"sync"
	"time"
)

type WarmUp struct {
	mu sync.Mutex

	cfg          *Config
	day          int
	dailyCount   int
	lastSend     time.Time
	growthFactor float64
	startedAt    time.Time
}

func NewWarmUp(cfg *Config) *WarmUp {
	return &WarmUp{
		cfg:          cfg,
		day:          1,
		growthFactor: 1.5 + rand.Float64()*0.7,
		startedAt:    time.Now(),
	}
}

func (wu *WarmUp) GetDailyLimit() int {
	wu.mu.Lock()
	defer wu.mu.Unlock()
	return wu.getDailyLimit()
}

func (wu *WarmUp) getDailyLimit() int {
	if wu.day >= wu.cfg.WarmUpDays {
		return wu.cfg.MaxPerDay
	}
	progress := float64(wu.day) / float64(wu.cfg.WarmUpDays)
	target := float64(wu.cfg.MaxPerDay-wu.cfg.InitialDailyLimit)*progress + float64(wu.cfg.InitialDailyLimit)
	limit := int(target * (1 + (wu.growthFactor-1.5)*0.1))
	if limit > wu.cfg.MaxPerDay {
		limit = wu.cfg.MaxPerDay
	}
	return limit
}

func (wu *WarmUp) CanSend() bool {
	wu.mu.Lock()
	defer wu.mu.Unlock()

	if time.Since(wu.lastSend) > wu.cfg.WarmUpInactivityTD {
		wu.day = 1
		wu.dailyCount = 0
		wu.startedAt = time.Now()
	}

	now := time.Now()
	if now.Day() != wu.startedAt.Day() || now.YearDay() != wu.startedAt.YearDay() {
		wu.day++
		wu.dailyCount = 0
		wu.startedAt = now
	}

	limit := wu.getDailyLimit()
	return wu.dailyCount < limit
}

func (wu *WarmUp) Record() {
	wu.mu.Lock()
	defer wu.mu.Unlock()
	wu.dailyCount++
	wu.lastSend = time.Now()
}

func (wu *WarmUp) GetStatus() map[string]any {
	wu.mu.Lock()
	defer wu.mu.Unlock()
	return map[string]any{
		"day":            wu.day,
		"total_days":     wu.cfg.WarmUpDays,
		"daily_count":    wu.dailyCount,
		"daily_limit":    wu.getDailyLimit(),
		"growth_factor":  wu.growthFactor,
		"started_at":     wu.startedAt,
		"inactive_hours": time.Since(wu.lastSend).Hours(),
	}
}

func (wu *WarmUp) ExportState() map[string]any {
	wu.mu.Lock()
	defer wu.mu.Unlock()
	return map[string]any{
		"day":            wu.day,
		"daily_count":    wu.dailyCount,
		"growth_factor":  wu.growthFactor,
		"started_at":     wu.startedAt,
	}
}

func (wu *WarmUp) ImportState(state map[string]any) {
	wu.mu.Lock()
	defer wu.mu.Unlock()
	if day, ok := state["day"].(int); ok {
		wu.day = day
	}
	if dc, ok := state["daily_count"].(int); ok {
		wu.dailyCount = dc
	}
	if gf, ok := state["growth_factor"].(float64); ok {
		wu.growthFactor = gf
	}
	if sa, ok := state["started_at"].(time.Time); ok {
		wu.startedAt = sa
	}
}

func (wu *WarmUp) Reset() {
	wu.mu.Lock()
	defer wu.mu.Unlock()
	wu.day = 1
	wu.dailyCount = 0
	wu.growthFactor = 1.5 + rand.Float64()*0.7
	wu.startedAt = time.Now()
}
