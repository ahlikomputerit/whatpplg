package antiban

import "time"

// Preset defines a named set of rate limits and safety parameters.
type Preset string

const (
	// PresetConservative is the safest preset: 2 msg/min, 30/hr, 150/day, 3-12s delay.
	PresetConservative Preset = "conservative"
	// PresetModerate is the default preset: 5 msg/min, 60/hr, 400/day, 1.5-8s delay.
	PresetModerate Preset = "moderate"
	// PresetAggressive allows higher throughput: 12 msg/min, 120/hr, 800/day, 0.5-4s delay.
	PresetAggressive Preset = "aggressive"
	// PresetHighVolume is for broadcast/high-traffic: 30 msg/min, 300/hr, 2000/day, 0.2-2s delay.
	PresetHighVolume Preset = "high-volume"
)

// Config holds all configurable parameters for the anti-ban system.
// Use DefaultConfig to create a preset-based config, then override fields as needed.
type Config struct {
	// Preset determines default values for all limits
	Preset Preset

	// RateLimiter
	MaxPerMinute         int
	MaxPerHour           int
	MaxPerDay            int
	MinDelayMs           int
	MaxDelayMs           int
	NewChatDelayMs       int
	MaxIdenticalMessages int
	BurstAllowance       int

	// WarmUp
	WarmUpDays         int
	InitialDailyLimit  int
	WarmUpInactivityTD time.Duration

	// Health
	AutoPauseRiskLevel int
	HealthDecayNormal  time.Duration
	HealthDecaySevere  time.Duration

	// TimelockGuard
	TimelockBlockDuration time.Duration

	// ReconnectThrottle
	ReconnectRampDuration time.Duration
	ReconnectInitialRate  float64

	// ContactGraph
	GroupLurkPeriod      time.Duration
	MaxStrangerPerDay    int
	HandshakeCooldown    time.Duration
	KnownCooldown        time.Duration

	// Scheduler
	ActiveHourStart int
	ActiveHourEnd   int
	WeekendFactor   float64
	PeakHourStart   int
	PeakHourEnd     int
	PeakBoost       float64

	// CircuitBreaker
	CircuitBreakerThreshold int
	CircuitBreakerCooldown  time.Duration

	// GroupGuard
	MaxGroupAddsPer10m    int
	MaxGroupRemovesPer10m int
	MaxGroupCreatesPer10m  int
	MaxGroupInvitesPer10m  int

	// ContentVariator
	EnableTypoInjection    bool
	TypoProbability        float64
	EnableZeroWidth        bool
	EnableEmojiPadding     bool
	EnablePunctuationVary  bool

	// DeliveryTracker
	DeliveryWindowSize   int
	DeliveryMinSamples   int
	DeliveryLowThreshold float64

	// ReplyRatio
	ReplyRatioMin         float64
	ReplyCooldownHours    int

	// Presence
	CircadianRhythm  string
	EnablePresence   bool

	// Recovery
	MaxBansBeforeHard int
	RecoveryRampPct   float64

	// InstanceCoordinator
	InstanceCount int
}

func (c *Config) applyPreset() {
	switch c.Preset {
	case PresetConservative:
		c.MaxPerMinute = 2
		c.MaxPerHour = 30
		c.MaxPerDay = 150
		c.MinDelayMs = 3000
		c.MaxDelayMs = 12000
		c.NewChatDelayMs = 15000
		c.BurstAllowance = 1
		c.WarmUpDays = 14
		c.InitialDailyLimit = 15
		c.MaxIdenticalMessages = 3
		c.AutoPauseRiskLevel = 30
		c.ReconnectInitialRate = 0.1
		c.ReconnectRampDuration = 120 * time.Second
		c.GroupLurkPeriod = 10 * time.Minute
		c.MaxStrangerPerDay = 5
		c.CircuitBreakerThreshold = 2
		c.CircuitBreakerCooldown = 30 * time.Minute
	case PresetModerate:
		c.MaxPerMinute = 5
		c.MaxPerHour = 60
		c.MaxPerDay = 400
		c.MinDelayMs = 1500
		c.MaxDelayMs = 8000
		c.NewChatDelayMs = 10000
		c.BurstAllowance = 2
		c.WarmUpDays = 10
		c.InitialDailyLimit = 30
		c.MaxIdenticalMessages = 5
		c.AutoPauseRiskLevel = 50
		c.ReconnectInitialRate = 0.25
		c.ReconnectRampDuration = 90 * time.Second
		c.GroupLurkPeriod = 5 * time.Minute
		c.MaxStrangerPerDay = 15
		c.CircuitBreakerThreshold = 3
		c.CircuitBreakerCooldown = 15 * time.Minute
	case PresetAggressive:
		c.MaxPerMinute = 12
		c.MaxPerHour = 120
		c.MaxPerDay = 800
		c.MinDelayMs = 500
		c.MaxDelayMs = 4000
		c.NewChatDelayMs = 5000
		c.BurstAllowance = 3
		c.WarmUpDays = 7
		c.InitialDailyLimit = 50
		c.MaxIdenticalMessages = 10
		c.AutoPauseRiskLevel = 70
		c.ReconnectInitialRate = 0.5
		c.ReconnectRampDuration = 60 * time.Second
		c.GroupLurkPeriod = 2 * time.Minute
		c.MaxStrangerPerDay = 40
		c.CircuitBreakerThreshold = 5
		c.CircuitBreakerCooldown = 5 * time.Minute
	case PresetHighVolume:
		c.MaxPerMinute = 30
		c.MaxPerHour = 300
		c.MaxPerDay = 2000
		c.MinDelayMs = 200
		c.MaxDelayMs = 2000
		c.NewChatDelayMs = 2000
		c.BurstAllowance = 5
		c.WarmUpDays = 5
		c.InitialDailyLimit = 100
		c.MaxIdenticalMessages = 20
		c.AutoPauseRiskLevel = 85
		c.ReconnectInitialRate = 0.7
		c.ReconnectRampDuration = 30 * time.Second
		c.GroupLurkPeriod = 0
		c.MaxStrangerPerDay = 100
		c.CircuitBreakerThreshold = 10
		c.CircuitBreakerCooldown = 1 * time.Minute
	}
}

// DefaultConfig returns a Config populated with defaults for the given preset.
// Each preset sets safe rate limits, delays, warmup days, and other parameters.
// Override individual fields after calling this function.
func DefaultConfig(preset Preset) Config {
	c := Config{
		Preset:                preset,
		ActiveHourStart:       8,
		ActiveHourEnd:         22,
		WeekendFactor:         0.6,
		PeakHourStart:         19,
		PeakHourEnd:           22,
		PeakBoost:             0.3,
		TimelockBlockDuration: 30 * time.Minute,
		WarmUpInactivityTD:    48 * time.Hour,
		HealthDecayNormal:     5 * time.Minute,
		HealthDecaySevere:     2 * time.Minute,
		DeliveryWindowSize:    100,
		DeliveryMinSamples:    10,
		DeliveryLowThreshold:  0.5,
		ReplyRatioMin:         0.1,
		ReplyCooldownHours:    48,
		CircadianRhythm:       "office",
		EnablePresence:        true,
		MaxBansBeforeHard:     3,
		RecoveryRampPct:       0.25,
		InstanceCount:         1,
		MaxGroupAddsPer10m:    3,
		MaxGroupRemovesPer10m: 5,
		MaxGroupCreatesPer10m:  2,
		MaxGroupInvitesPer10m: 10,
		EnableTypoInjection:   false,
		TypoProbability:       0.025,
	}
	c.applyPreset()
	return c
}

// ContactState tracks the relationship level with a contact.
type ContactState int

const (
	// ContactStranger means no prior interaction.
	ContactStranger ContactState = iota
	// ContactHandshakeSent means a handshake message was sent.
	ContactHandshakeSent
	// ContactHandshakeComplete means the handshake was replied to.
	ContactHandshakeComplete
	// ContactKnown means the contact is whitelisted for normal sending.
	ContactKnown
)

// RiskLevel indicates the current account risk assessment.
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

// BanType categorises the kind of ban detected.
type BanType string

const (
	BanTimelock      BanType = "timelock"
	BanRateOverlimit  BanType = "rate_overlimit"
	BanSoft           BanType = "soft_ban"
	BanHard           BanType = "hard_ban"
)

// RecoveryPhase tracks the phase of a ban recovery cycle.
type RecoveryPhase string

const (
	PhasePaused     RecoveryPhase = "paused"
	PhaseRecovering RecoveryPhase = "recovering"
	PhaseRamping    RecoveryPhase = "ramping"
	PhaseGraduated  RecoveryPhase = "graduated"
	PhaseDead       RecoveryPhase = "dead"
)
