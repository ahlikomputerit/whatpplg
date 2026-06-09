package antiban

import (
	"sync"
	"time"
)

// GroupOperation represents a type of group management operation.
type GroupOperation int

const (
	OpGroupAdd    GroupOperation = iota
	OpGroupRemove
	OpGroupCreate
	OpGroupInvite
)

// GroupOperationGuard rate-limits group operations (add, remove, create, invite) per 10-minute window.
type GroupOperationGuard struct {
	mu sync.Mutex

	cfg  *Config
	ops  map[string][]time.Time
}

// NewGroupOperationGuard creates a new group operation guard.
func NewGroupOperationGuard(cfg *Config) *GroupOperationGuard {
	return &GroupOperationGuard{
		cfg: cfg,
		ops: make(map[string][]time.Time),
	}
}

// Check returns whether the given group operation is allowed under current rate limits.
func (gog *GroupOperationGuard) Check(op GroupOperation) bool {
	gog.mu.Lock()
	defer gog.mu.Unlock()

	var key string
	var limit int
	switch op {
	case OpGroupAdd:
		key = "add"
		limit = gog.cfg.MaxGroupAddsPer10m
	case OpGroupRemove:
		key = "remove"
		limit = gog.cfg.MaxGroupRemovesPer10m
	case OpGroupCreate:
		key = "create"
		limit = gog.cfg.MaxGroupCreatesPer10m
	case OpGroupInvite:
		key = "invite"
		limit = gog.cfg.MaxGroupInvitesPer10m
	default:
		return false
	}

	cutoff := time.Now().Add(-10 * time.Minute)
	ops := gog.ops[key]
	i := 0
	for _, t := range ops {
		if t.After(cutoff) {
			ops[i] = t
			i++
		}
	}
	ops = ops[:i]

	if len(ops) >= limit {
		gog.ops[key] = ops
		return false
	}

	ops = append(ops, time.Now())
	gog.ops[key] = ops
	return true
}

// Reset clears all group operation tracking data.
func (gog *GroupOperationGuard) Reset() {
	gog.mu.Lock()
	defer gog.mu.Unlock()
	gog.ops = make(map[string][]time.Time)
}

// GetStats returns group operation counts per type for monitoring.
func (gog *GroupOperationGuard) GetStats() map[string]any {
	gog.mu.Lock()
	defer gog.mu.Unlock()

	cutoff := time.Now().Add(-10 * time.Minute)
	stats := make(map[string]any)
	for key, ops := range gog.ops {
		count := 0
		for _, t := range ops {
			if t.After(cutoff) {
				count++
			}
		}
		stats[key] = count
	}
	return stats
}
