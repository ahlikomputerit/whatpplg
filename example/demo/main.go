// Demo program untuk whatsmeow-antiban.
// Menjalankan simulasi anti-ban tanpa koneksi WhatsApp.
//
// Jalankan:
//   go run ./example/demo.go
package main

import (
	"fmt"
	"time"

	antiban "github.com/ahlikomputerit/whatpplg"
)

func main() {
	fmt.Println("=== whatsmeow-antiban Demo ===")
	fmt.Println()

	// 1. Demo Presets
	demoPresets()

	// 2. Demo RateLimiter
	demoRateLimiter()

	// 3. Demo WarmUp
	demoWarmUp()

	// 4. Demo Health Monitor
	demoHealth()

	// 5. Demo Content Variator
	demoContentVariator()

	// 6. Demo Device Fingerprint
	demoDeviceFingerprint()

	// 7. Demo Circuit Breaker
	demoCircuitBreaker()

	// 8. Demo Ban Recovery
	demoBanRecovery()

	// 9. Demo Scheduler
	demoScheduler()

	// 10. Demo AntiBan Orchestrator
	demoOrchestrator()

	fmt.Println("=== Demo Selesai ===")
}

func demoPresets() {
	fmt.Println("--- Presets ---")
	for _, p := range []antiban.Preset{
		antiban.PresetConservative,
		antiban.PresetModerate,
		antiban.PresetAggressive,
		antiban.PresetHighVolume,
	} {
		cfg := antiban.DefaultConfig(p)
		fmt.Printf("  %-15s max/min:%-2d max/hr:%-4d max/day:%-5d minDelay:%-4dms\n",
			p, cfg.MaxPerMinute, cfg.MaxPerHour, cfg.MaxPerDay, cfg.MinDelayMs)
	}
	fmt.Println()
}

func demoRateLimiter() {
	fmt.Println("--- Rate Limiter ---")
	cfg := antiban.DefaultConfig(antiban.PresetConservative)
	rl := antiban.NewRateLimiter(&cfg)

	fmt.Println("  Mengirim 10 pesan berturut-turut:")
	for i := 1; i <= 10; i++ {
		canSend := rl.CanSend()
		delay := rl.GetDelay(fmt.Sprintf("user%d@s.whatsapp.net", i%3), []byte(fmt.Sprintf("Pesan %d", i)))
		allowed := "✅"
		if !canSend {
			allowed = "❌"
		}
		fmt.Printf("    Pesan #%-2d delay:%-6v %s\n", i, delay.Round(time.Millisecond), allowed)
		if canSend {
			rl.Record(fmt.Sprintf("user%d@s.whatsapp.net", i%3))
		}
	}

	stats := rl.GetStats()
	fmt.Printf("  Statistik: %d terkirim, %d known chats\n", stats["sent"], stats["known_chats"])
	fmt.Println()
}

func demoWarmUp() {
	fmt.Println("--- WarmUp ---")
	cfg := antiban.DefaultConfig(antiban.PresetConservative)
	cfg.WarmUpDays = 7
	cfg.InitialDailyLimit = 10
	cfg.MaxPerDay = 100
	w := antiban.NewWarmUp(&cfg)

	fmt.Printf("  Hari ke-%d, limit: %d pesan/hari\n", 1, w.GetDailyLimit())
	for i := 1; i <= 12; i++ {
		if w.CanSend() {
			w.Record()
			fmt.Printf("    ✅ Pesan #%d terkirim\n", i)
		} else {
			fmt.Printf("    ❌ Pesan #%d diblokir (limit harian)\n", i)
			break
		}
	}
	fmt.Println()
}

func demoHealth() {
	fmt.Println("--- Health Monitor ---")
	cfg := antiban.DefaultConfig(antiban.PresetConservative)
	h := antiban.NewHealthMonitor(&cfg)

	fmt.Println("  Score awal:", h.GetStatus()["score"])

	h.RecordDisconnect()
	fmt.Println("  Setelah disconnect:", h.GetStatus()["score"], h.GetRiskLevel())

	h.RecordForbidden()
	fmt.Println("  Setelah forbidden:", h.GetStatus()["score"], h.GetRiskLevel())

	h.RecordLoggedOut()
	fmt.Println("  Setelah logged out:", h.GetStatus()["score"], h.GetRiskLevel())

	h.RecordReconnect()
	fmt.Println("  Setelah reconnect:", h.GetStatus()["score"], h.GetRiskLevel())

	fmt.Println("  Auto-paused:", h.IsPaused())
	fmt.Println()
}

func demoContentVariator() {
	fmt.Println("--- Content Variator ---")
	cfg := antiban.DefaultConfig(antiban.PresetModerate)
	cfg.EnableTypoInjection = true
	cfg.TypoProbability = 1.0
	cfg.EnableZeroWidth = true
	cfg.EnableEmojiPadding = true
	cfg.EnablePunctuationVary = true
	cv := antiban.NewContentVariator(&cfg)

	original := "Hello, how are you today?"
	fmt.Printf("  Original: %q\n", original)
	for i := 1; i <= 5; i++ {
		varied := cv.Vary(original)
		fmt.Printf("  Variasi #%d: %q\n", i, varied)
	}
	fmt.Println()
}

func demoDeviceFingerprint() {
	fmt.Println("--- Device Fingerprint ---")
	fp := antiban.GenerateFingerprint()
	fmt.Printf("  App: %s\n", fp.AppVersion)
	fmt.Printf("  OS:  %s %s\n", fp.OS, fp.OsVersion)
	fmt.Printf("  Device: %s\n", fp.DeviceModel)
	fmt.Printf("  User-Agent: %s\n", fp.UserAgent())

	fp2 := antiban.FingerprintFromString("test-user-123")
	fmt.Printf("  Fingerprint dari string: %s / %s\n", fp2.AppVersion, fp2.DeviceModel)
	fmt.Println()
}

func demoCircuitBreaker() {
	fmt.Println("--- Circuit Breaker ---")
	cfg := antiban.DefaultConfig(antiban.PresetConservative)
	cfg.CircuitBreakerThreshold = 2
	cfg.CircuitBreakerCooldown = 100 * time.Millisecond
	cb := antiban.NewJidCircuitBreaker(&cfg)

	jid := "problem-user@s.whatsapp.net"
	fmt.Printf("  JID: %s\n", jid)

	fmt.Printf("  Awal: CanSend=%t\n", cb.CanSend(jid))
	cb.RecordFailure(jid)
	fmt.Printf("  Gagal 1x: CanSend=%t\n", cb.CanSend(jid))
	cb.RecordFailure(jid)
	fmt.Printf("  Gagal 2x: CanSend=%t (circuit OPEN)\n", cb.CanSend(jid))
	_ = cb.GetJitter(jid)

	time.Sleep(150 * time.Millisecond)
	fmt.Printf("  Setelah cooldown: CanSend=%t (half-open)\n", cb.CanSend(jid))

	cb.RecordSuccess(jid)
	fmt.Printf("  Sukses: CanSend=%t (closed)\n", cb.CanSend(jid))
	fmt.Println()
}

func demoBanRecovery() {
	fmt.Println("--- Ban Recovery ---")
	cfg := antiban.DefaultConfig(antiban.PresetConservative)
	cfg.MaxBansBeforeHard = 3
	cfg.RecoveryRampPct = 0.25
	bro := antiban.NewBanRecoveryOrchestrator(&cfg)

	fmt.Printf("  Awal: fase=%s multiplier=%.2f\n", bro.GetStatus()["phase"], bro.GetRateMultiplier())

	bro.RecordBanEvent(antiban.BanEvent{Type: antiban.BanTimelock, Timestamp: time.Now()})
	fmt.Printf("  Timelock: fase=%s multiplier=%.2f\n", bro.GetStatus()["phase"], bro.GetRateMultiplier())

	bro.Tick()
	fmt.Printf("  Tick 1 (recovering): fase=%s multiplier=%.2f\n", bro.GetStatus()["phase"], bro.GetRateMultiplier())

	bro.Tick()
	fmt.Printf("  Tick 2 (ramping): fase=%s multiplier=%.2f\n", bro.GetStatus()["phase"], bro.GetRateMultiplier())

	for i := 0; i < 7; i++ {
		bro.Tick()
	}
	fmt.Printf("  Setelah ramp selesai: fase=%s multiplier=%.2f\n", bro.GetStatus()["phase"], bro.GetRateMultiplier())
	fmt.Println()
}

func demoScheduler() {
	fmt.Println("--- Scheduler ---")
	cfg := antiban.DefaultConfig(antiban.PresetConservative)
	s := antiban.NewScheduler(&cfg)

	status := s.GetStatus()
	fmt.Printf("  Active now: %t\n", status["active"])
	fmt.Printf("  Speed factor: %.2f\n", status["speed"])
	fmt.Printf("  Ms until active: %d\n", status["next_ms"])
	fmt.Println()
}

func demoOrchestrator() {
	fmt.Println("--- AntiBan Orchestrator ---")
	cfg := antiban.DefaultConfig(antiban.PresetConservative)
	cfg.MinDelayMs = 10
	cfg.MaxDelayMs = 50
	cfg.WarmUpDays = 1
	cfg.InitialDailyLimit = 5
	cfg.MaxPerDay = 10
	cfg.MaxPerMinute = 3
	cfg.GroupLurkPeriod = 0
	ab := antiban.New(antiban.PresetConservative, cfg)

	fmt.Println("  Mengirim 6 pesan ke individual chat:")
	for i := 1; i <= 6; i++ {
		delay, allowed := ab.BeforeSend(fmt.Sprintf("user%d@s.whatsapp.net", i%2), []byte(fmt.Sprintf("Pesan %d", i)))
		status := "✅"
		if !allowed {
			status = "❌"
		}
		fmt.Printf("    Pesan #%-2d delay:%-6v %s\n", i, delay.Round(time.Millisecond), status)
		if allowed {
			ab.AfterSend(fmt.Sprintf("user%d@s.whatsapp.net", i%2), true)
		}
	}

	stats := ab.GetStats()
	fmt.Printf("  Paused: %t\n", stats["paused"])
	fmt.Printf("  Health risk: %v\n", stats["health"].(map[string]any)["risk_level"])
	fmt.Printf("  Rate limiter sent: %d\n", stats["rate_limiter"].(map[string]any)["sent"])
	fmt.Printf("  Warmup day: %d\n", stats["warmup"].(map[string]any)["day"])
	fmt.Println()
}
