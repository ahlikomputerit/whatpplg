package antiban

import "strings"

func IsGroup(jid string) bool {
	return strings.Contains(jid, "@g.us")
}

func IsNewsletter(jid string) bool {
	return strings.Contains(jid, "@newsletter")
}

func IsBroadcast(jid string) bool {
	return strings.Contains(jid, "@broadcast")
}

func ShouldUseGroupProfile(jid string) bool {
	return IsGroup(jid) || IsNewsletter(jid)
}

func ApplyGroupMultiplier[T int | float64](val T, multiplier float64) T {
	switch v := any(val).(type) {
	case int:
		return T(int(float64(v) * multiplier))
	case float64:
		return T(v * multiplier)
	default:
		return val
	}
}
