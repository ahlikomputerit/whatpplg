package antiban

import (
	"strings"
	"sync"
	"time"
)

type RetryReason int

const (
	RetryReasonUnknown      RetryReason = 0
	RetryReasonNoSession    RetryReason = 1
	RetryReasonInvalidKey   RetryReason = 2
	RetryReasonBadMac       RetryReason = 4
	RetryReasonTooOld       RetryReason = 5
	RetryReasonTimeout      RetryReason = 6
	RetryReasonGeneric      RetryReason = 7
	RetryReasonMediaRetry   RetryReason = 8
	RetryReasonSessionReset RetryReason = 9
)

type RetryReasonTracker struct {
	mu sync.Mutex

	retries   map[string][]RetryReason
	spiralThreshold int
}

func NewRetryReasonTracker(spiralThreshold int) *RetryReasonTracker {
	if spiralThreshold <= 0 {
		spiralThreshold = 5
	}
	return &RetryReasonTracker{
		retries:         make(map[string][]RetryReason),
		spiralThreshold: spiralThreshold,
	}
}

func (rrt *RetryReasonTracker) Classify(reasonCode int) RetryReason {
	switch reasonCode {
	case 1:
		return RetryReasonNoSession
	case 2:
		return RetryReasonInvalidKey
	case 4, 7:
		return RetryReasonBadMac
	case 5:
		return RetryReasonTooOld
	case 6:
		return RetryReasonTimeout
	case 8:
		return RetryReasonMediaRetry
	case 9:
		return RetryReasonSessionReset
	default:
		return RetryReasonGeneric
	}
}

func (rrt *RetryReasonTracker) ClassifyFromError(err error) RetryReason {
	if err == nil {
		return RetryReasonUnknown
	}
	errStr := err.Error()
	switch {
	case strings.Contains(errStr, "no session") || strings.Contains(errStr, "NoSession"):
		return RetryReasonNoSession
	case strings.Contains(errStr, "invalid key") || strings.Contains(errStr, "InvalidKey"):
		return RetryReasonInvalidKey
	case strings.Contains(errStr, "bad mac") || strings.Contains(errStr, "BadMac") || strings.Contains(errStr, "MAC"):
		return RetryReasonBadMac
	case strings.Contains(errStr, "timeout"):
		return RetryReasonTimeout
	case strings.Contains(errStr, "media"):
		return RetryReasonMediaRetry
	default:
		return RetryReasonGeneric
	}
}

func (rrt *RetryReasonTracker) RecordRetry(msgID string, reason RetryReason) {
	rrt.mu.Lock()
	defer rrt.mu.Unlock()
	rrt.retries[msgID] = append(rrt.retries[msgID], reason)

	go func() {
		time.Sleep(15 * time.Minute)
		rrt.mu.Lock()
		delete(rrt.retries, msgID)
		rrt.mu.Unlock()
	}()
}

func (rrt *RetryReasonTracker) IsSpiraling(msgID string) bool {
	rrt.mu.Lock()
	defer rrt.mu.Unlock()

	reasons, ok := rrt.retries[msgID]
	if !ok {
		return false
	}

	return len(reasons) >= rrt.spiralThreshold
}

func (rrt *RetryReasonTracker) Clear() {
	rrt.mu.Lock()
	defer rrt.mu.Unlock()
	rrt.retries = make(map[string][]RetryReason)
}

func (rrt *RetryReasonTracker) GetStats() map[string]any {
	rrt.mu.Lock()
	defer rrt.mu.Unlock()

	stats := make(map[RetryReason]int)
	for _, reasons := range rrt.retries {
		for _, r := range reasons {
			stats[r]++
		}
	}

	result := make(map[string]any)
	for k, v := range stats {
		switch k {
		case RetryReasonNoSession:
			result["no_session"] = v
		case RetryReasonInvalidKey:
			result["invalid_key"] = v
		case RetryReasonBadMac:
			result["bad_mac"] = v
		case RetryReasonTimeout:
			result["timeout"] = v
		case RetryReasonMediaRetry:
			result["media_retry"] = v
		case RetryReasonSessionReset:
			result["session_reset"] = v
		default:
			result["other"] = v
		}
	}
	result["total"] = len(rrt.retries)
	return result
}

func (rrt *RetryReasonTracker) Destroy() {
	rrt.Clear()
}
