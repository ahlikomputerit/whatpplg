package antiban

import (
	"testing"
	"time"
)

func TestProxyRotator_New(t *testing.T) {
	eps := []*ProxyEndpoint{
		{URL: "socks5://proxy1:1080", Weight: 1},
		{URL: "socks5://proxy2:1080", Weight: 2},
	}
	pr := NewProxyRotator(eps, RotateRoundRobin)
	if pr == nil {
		t.Fatal("expected non-nil ProxyRotator")
	}
}

func TestProxyRotator_EmptyEndpoints(t *testing.T) {
	pr := NewProxyRotator(nil, RotateRoundRobin)
	if pr.Current() != nil {
		t.Fatal("expected nil current for empty endpoints")
	}
	u := pr.Rotate()
	if u != nil {
		t.Fatal("expected nil from rotate on empty endpoints")
	}
}

func TestProxyRotator_RoundRobin(t *testing.T) {
	eps := []*ProxyEndpoint{
		{URL: "socks5://proxy1:1080", Weight: 1},
		{URL: "socks5://proxy2:1080", Weight: 1},
	}
	pr := NewProxyRotator(eps, RotateRoundRobin)

	u1 := pr.Rotate()
	u2 := pr.Rotate()
	u3 := pr.Rotate()

	if u1.String() == u2.String() {
		t.Fatal("expected different proxies on round-robin rotation")
	}
	if u3.String() != u1.String() {
		t.Fatalf("expected wrap-around to first proxy after full cycle")
	}
}

func TestProxyRotator_Random(t *testing.T) {
	eps := []*ProxyEndpoint{
		{URL: "socks5://proxy1:1080", Weight: 1},
		{URL: "socks5://proxy2:1080", Weight: 1},
		{URL: "socks5://proxy3:1080", Weight: 1},
	}
	pr := NewProxyRotator(eps, RotateRandom)

	seen := make(map[string]bool)
	for i := 0; i < 10; i++ {
		u := pr.Rotate()
		seen[u.String()] = true
	}
	if len(seen) < 2 {
		t.Log("note: random may pick same proxy repeatedly")
	}
}

func TestProxyRotator_Current(t *testing.T) {
	eps := []*ProxyEndpoint{
		{URL: "socks5://proxy1:1080", Weight: 1},
	}
	pr := NewProxyRotator(eps, RotateRoundRobin)
	u := pr.Current()
	if u == nil || u.String() != "socks5://proxy1:1080" {
		t.Fatalf("expected socks5://proxy1:1080, got %v", u)
	}
}

func TestProxyRotator_MarkFailure(t *testing.T) {
	eps := []*ProxyEndpoint{
		{URL: "socks5://proxy1:1080", Weight: 1},
	}
	pr := NewProxyRotator(eps, RotateRoundRobin)
	pr.MarkFailure("socks5://proxy1:1080")
	pr.MarkFailure("socks5://proxy1:1080")
	pr.MarkFailure("socks5://proxy1:1080")

	if eps[0].FailureCount != 3 {
		t.Fatalf("expected 3 failures, got %d", eps[0].FailureCount)
	}
	if eps[0].Cooldown <= 0 {
		t.Fatal("expected cooldown after 3 failures")
	}
}

func TestProxyRotator_ResurrectAll(t *testing.T) {
	eps := []*ProxyEndpoint{
		{URL: "socks5://proxy1:1080", Weight: 1},
	}
	pr := NewProxyRotator(eps, RotateRoundRobin)
	pr.MarkFailure("socks5://proxy1:1080")
	pr.ResurrectAll()
	if eps[0].FailureCount != 0 {
		t.Fatalf("expected 0 failures after resurrect, got %d", eps[0].FailureCount)
	}
}

func TestProxyRotator_StartStop(t *testing.T) {
	eps := []*ProxyEndpoint{
		{URL: "socks5://proxy1:1080", Weight: 1},
	}
	pr := NewProxyRotator(eps, RotateRoundRobin)
	pr.StartRotateEvery(10 * time.Millisecond)
	time.Sleep(15 * time.Millisecond)
	pr.Stop()
}

func TestProxyRotator_GetStats(t *testing.T) {
	eps := []*ProxyEndpoint{
		{URL: "socks5://proxy1:1080", Weight: 1},
	}
	pr := NewProxyRotator(eps, RotateRoundRobin)
	stats := pr.GetStats()
	if stats["rotations"].(int64) != 0 {
		t.Fatalf("expected 0 rotations, got %d", stats["rotations"])
	}
	if stats["total"] != nil {
		t.Log("note: stats has total key")
	}
}

func TestItoa(t *testing.T) {
	if itoa(0) != "0" {
		t.Fatalf("expected 0, got %s", itoa(0))
	}
	if itoa(42) != "42" {
		t.Fatalf("expected 42, got %s", itoa(42))
	}
}

func TestMax(t *testing.T) {
	if max(5, 3) != 5 {
		t.Fatal("expected 5")
	}
	if max(3, 5) != 5 {
		t.Fatal("expected 5")
	}
	if max(5, 5) != 5 {
		t.Fatal("expected 5")
	}
}
