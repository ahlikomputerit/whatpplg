package antiban

import (
	"math/rand/v2"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type ProxyEndpoint struct {
	URL          string
	Weight       int
	LastUsed     time.Time
	FailureCount int
	Cooldown     time.Duration
}

type RotateStrategy int

const (
	RotateRoundRobin RotateStrategy = iota
	RotateRandom
	RotateLeastRecentlyUsed
	RotateWeighted
)

type ProxyRotator struct {
	mu sync.Mutex

	cfg       *Config
	endpoints []*ProxyEndpoint
	current   int
	strategy  RotateStrategy
	ticker    *time.Ticker
	stopCh    chan struct{}

	rotateCount atomic.Int64
}

func NewProxyRotator(endpoints []*ProxyEndpoint, strategy RotateStrategy) *ProxyRotator {
	pr := &ProxyRotator{
		endpoints: endpoints,
		strategy:  strategy,
		stopCh:    make(chan struct{}),
	}
	return pr
}

func (pr *ProxyRotator) StartRotateEvery(interval time.Duration) {
	pr.mu.Lock()
	if pr.ticker != nil {
		pr.ticker.Stop()
	}
	pr.ticker = time.NewTicker(interval)
	pr.mu.Unlock()

	go func() {
		for {
			select {
			case <-pr.ticker.C:
				pr.Rotate()
			case <-pr.stopCh:
				return
			}
		}
	}()
}

func (pr *ProxyRotator) Stop() {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	if pr.ticker != nil {
		pr.ticker.Stop()
		pr.ticker = nil
	}
	close(pr.stopCh)
}

func (pr *ProxyRotator) Current() *url.URL {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	return pr.getCurrent()
}

func (pr *ProxyRotator) getCurrent() *url.URL {
	if len(pr.endpoints) == 0 {
		return nil
	}
	ep := pr.endpoints[pr.current]
	u, err := url.Parse(ep.URL)
	if err != nil {
		return nil
	}
	return u
}

func (pr *ProxyRotator) Rotate() *url.URL {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if len(pr.endpoints) == 0 {
		return nil
	}

	switch pr.strategy {
	case RotateRoundRobin:
		pr.current = (pr.current + 1) % len(pr.endpoints)
	case RotateRandom:
		pr.current = rand.IntN(len(pr.endpoints))
	case RotateLeastRecentlyUsed:
		pr.current = pr.findLRU()
	case RotateWeighted:
		pr.current = pr.findWeighted()
	}

	ep := pr.endpoints[pr.current]
	ep.LastUsed = time.Now()
	pr.rotateCount.Add(1)

	u, err := url.Parse(ep.URL)
	if err != nil {
		return nil
	}
	return u
}

func (pr *ProxyRotator) findLRU() int {
	if len(pr.endpoints) == 0 {
		return 0
	}
	idx := 0
	oldest := pr.endpoints[0].LastUsed
	for i, ep := range pr.endpoints {
		if ep.LastUsed.Before(oldest) && ep.FailureCount < 3 {
			oldest = ep.LastUsed
			idx = i
		}
	}
	return idx
}

func (pr *ProxyRotator) findWeighted() int {
	totalWeight := 0
	for _, ep := range pr.endpoints {
		if time.Since(ep.LastUsed) > ep.Cooldown || ep.Cooldown == 0 {
			totalWeight += max(1, ep.Weight-ep.FailureCount*2)
		}
	}
	if totalWeight == 0 {
		return 0
	}

	r := rand.IntN(totalWeight)
	for i, ep := range pr.endpoints {
		w := max(1, ep.Weight-ep.FailureCount*2)
		if r < w {
			return i
		}
		r -= w
	}
	return 0
}

func (pr *ProxyRotator) MarkFailure(endpointURL string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	for _, ep := range pr.endpoints {
		if ep.URL == endpointURL {
			ep.FailureCount++
			if ep.FailureCount >= 3 {
				ep.Cooldown = 5 * time.Minute
			}
			return
		}
	}
}

func (pr *ProxyRotator) ResurrectAll() {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	for _, ep := range pr.endpoints {
		ep.FailureCount = 0
	}
}

func (pr *ProxyRotator) GetStats() map[string]any {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	stats := make(map[string]any)
	for i, ep := range pr.endpoints {
		stats["endpoint_"+itoa(i)] = map[string]any{
			"url":     ep.URL,
			"failures": ep.FailureCount,
			"last_used": ep.LastUsed,
		}
	}
	stats["current"] = pr.current
	stats["rotations"] = pr.rotateCount.Load()
	return stats
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
