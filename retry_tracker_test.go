package antiban

import (
	"errors"
	"testing"
)

func TestRetryReasonTracker_New(t *testing.T) {
	rrt := NewRetryReasonTracker(5)
	if rrt == nil {
		t.Fatal("expected non-nil RetryReasonTracker")
	}
}

func TestRetryReasonTracker_Classify(t *testing.T) {
	rrt := NewRetryReasonTracker(5)

	tests := []struct {
		code     int
		expected RetryReason
	}{
		{1, RetryReasonNoSession},
		{2, RetryReasonInvalidKey},
		{4, RetryReasonBadMac},
		{7, RetryReasonBadMac},
		{5, RetryReasonTooOld},
		{6, RetryReasonTimeout},
		{8, RetryReasonMediaRetry},
		{9, RetryReasonSessionReset},
		{99, RetryReasonGeneric},
	}

	for _, tc := range tests {
		got := rrt.Classify(tc.code)
		if got != tc.expected {
			t.Errorf("Classify(%d) = %v, want %v", tc.code, got, tc.expected)
		}
	}
}

func TestRetryReasonTracker_ClassifyFromError(t *testing.T) {
	rrt := NewRetryReasonTracker(5)

	tests := []struct {
		err      error
		expected RetryReason
	}{
		{nil, RetryReasonUnknown},
		{errors.New("no session"), RetryReasonNoSession},
		{errors.New("invalid key"), RetryReasonInvalidKey},
		{errors.New("bad mac"), RetryReasonBadMac},
		{errors.New("MAC error"), RetryReasonBadMac},
		{errors.New("timeout"), RetryReasonTimeout},
		{errors.New("media error"), RetryReasonMediaRetry},
		{errors.New("something else"), RetryReasonGeneric},
	}

	for _, tc := range tests {
		got := rrt.ClassifyFromError(tc.err)
		if got != tc.expected {
			t.Errorf("ClassifyFromError(%v) = %v, want %v", tc.err, got, tc.expected)
		}
	}
}

func TestRetryReasonTracker_RecordAndIsSpiraling(t *testing.T) {
	rrt := NewRetryReasonTracker(3)

	rrt.RecordRetry("msg1", RetryReasonBadMac)
	if rrt.IsSpiraling("msg1") {
		t.Fatal("expected not spiraling after 1 retry")
	}

	rrt.RecordRetry("msg1", RetryReasonBadMac)
	rrt.RecordRetry("msg1", RetryReasonBadMac)
	if !rrt.IsSpiraling("msg1") {
		t.Fatal("expected spiraling after 3 retries")
	}
}

func TestRetryReasonTracker_NonExistentMessage(t *testing.T) {
	rrt := NewRetryReasonTracker(5)
	if rrt.IsSpiraling("nonexistent") {
		t.Fatal("expected not spiraling for non-existent message")
	}
}

func TestRetryReasonTracker_Clear(t *testing.T) {
	rrt := NewRetryReasonTracker(3)
	rrt.RecordRetry("msg1", RetryReasonBadMac)
	rrt.Clear()
	if rrt.IsSpiraling("msg1") {
		t.Fatal("expected not spiraling after clear")
	}
}

func TestRetryReasonTracker_GetStats(t *testing.T) {
	rrt := NewRetryReasonTracker(5)
	rrt.RecordRetry("msg1", RetryReasonBadMac)
	rrt.RecordRetry("msg1", RetryReasonTimeout)
	rrt.RecordRetry("msg2", RetryReasonNoSession)

	stats := rrt.GetStats()
	if stats["total"].(int) != 2 {
		t.Fatalf("expected total 2, got %d", stats["total"])
	}
	if stats["bad_mac"].(int) != 1 {
		t.Fatalf("expected 1 bad_mac, got %d", stats["bad_mac"])
	}
	if stats["timeout"].(int) != 1 {
		t.Fatalf("expected 1 timeout, got %d", stats["timeout"])
	}
	if stats["no_session"].(int) != 1 {
		t.Fatalf("expected 1 no_session, got %d", stats["no_session"])
	}
}

func TestRetryReasonTracker_Destroy(t *testing.T) {
	rrt := NewRetryReasonTracker(5)
	rrt.RecordRetry("msg1", RetryReasonBadMac)
	rrt.Destroy()
	stats := rrt.GetStats()
	if stats["total"].(int) != 0 {
		t.Fatalf("expected total 0 after destroy, got %d", stats["total"])
	}
}

func TestNewRetryReasonTracker_DefaultThreshold(t *testing.T) {
	rrt := NewRetryReasonTracker(0)
	if rrt.spiralThreshold != 5 {
		t.Fatalf("expected default threshold 5, got %d", rrt.spiralThreshold)
	}
}
