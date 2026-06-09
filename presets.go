package antiban

// ResolveConfig applies a preset and then overlays any non-zero overrides.
// This allows partial configs to customise specific limits while keeping
// the rest of the preset defaults.
func ResolveConfig(preset Preset, overrides ...Config) Config {
	cfg := DefaultConfig(preset)
	if len(overrides) > 0 {
		o := overrides[0]

		cfg.Preset = preset
		cfg.applyPreset()

		if o.MaxPerMinute > 0 {
			cfg.MaxPerMinute = o.MaxPerMinute
		}
		if o.MaxPerHour > 0 {
			cfg.MaxPerHour = o.MaxPerHour
		}
		if o.MaxPerDay > 0 {
			cfg.MaxPerDay = o.MaxPerDay
		}
		if o.MinDelayMs > 0 {
			cfg.MinDelayMs = o.MinDelayMs
		}
		if o.MaxDelayMs > 0 {
			cfg.MaxDelayMs = o.MaxDelayMs
		}
		if o.NewChatDelayMs > 0 {
			cfg.NewChatDelayMs = o.NewChatDelayMs
		}
		if o.WarmUpDays > 0 {
			cfg.WarmUpDays = o.WarmUpDays
		}
		if o.InitialDailyLimit > 0 {
			cfg.InitialDailyLimit = o.InitialDailyLimit
		}
		if o.MaxIdenticalMessages > 0 {
			cfg.MaxIdenticalMessages = o.MaxIdenticalMessages
		}
		if o.BurstAllowance > 0 {
			cfg.BurstAllowance = o.BurstAllowance
		}
		if o.AutoPauseRiskLevel > 0 {
			cfg.AutoPauseRiskLevel = o.AutoPauseRiskLevel
		}
		if o.ReconnectRampDuration > 0 {
			cfg.ReconnectRampDuration = o.ReconnectRampDuration
		}
		if o.ReconnectInitialRate > 0 {
			cfg.ReconnectInitialRate = o.ReconnectInitialRate
		}
		if o.GroupLurkPeriod > 0 {
			cfg.GroupLurkPeriod = o.GroupLurkPeriod
		}
		if o.MaxStrangerPerDay > 0 {
			cfg.MaxStrangerPerDay = o.MaxStrangerPerDay
		}
		if o.ActiveHourStart > 0 {
			cfg.ActiveHourStart = o.ActiveHourStart
		}
		if o.ActiveHourEnd > 0 {
			cfg.ActiveHourEnd = o.ActiveHourEnd
		}
		if o.CircuitBreakerThreshold > 0 {
			cfg.CircuitBreakerThreshold = o.CircuitBreakerThreshold
		}
		if o.CircuitBreakerCooldown > 0 {
			cfg.CircuitBreakerCooldown = o.CircuitBreakerCooldown
		}
		if o.EnableTypoInjection {
			cfg.EnableTypoInjection = true
		}
		if o.TypoProbability > 0 {
			cfg.TypoProbability = o.TypoProbability
		}
		if o.EnableZeroWidth {
			cfg.EnableZeroWidth = true
		}
		if o.EnableEmojiPadding {
			cfg.EnableEmojiPadding = true
		}
	}
	return cfg
}
