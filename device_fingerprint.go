package antiban

import (
	"math/rand/v2"
)

// DeviceFingerprint represents a WhatsApp client device identity.
type DeviceFingerprint struct {
	AppVersion   string
	OsVersion    string
	DeviceModel  string
	OS           string
}

var appVersions = []string{
	"2.24.25", "2.24.26", "2.25.1", "2.25.2", "2.25.3",
	"2.24.20", "2.24.22", "2.24.23", "2.24.24",
}

var deviceModels = []string{
	"iPhone14,3", "iPhone15,2", "iPhone15,3", "iPhone15,4",
	"Pixel 7", "Pixel 7 Pro", "Pixel 8", "Pixel 8 Pro",
	"SM-S908B", "SM-S901B", "SM-S911B", "SM-S921B",
	"SM-G998B", "SM-G996B", "SM-G990E",
	"2107113SG", "2201123C", "2201122G", "2211133G",
}

var osVersions = []string{
	"17.0", "17.1", "17.2", "17.3", "17.4", "17.5",
	"14", "15", "13",
	"12.0", "13.0", "14.0",
}

var oss = []string{
	"iOS", "Android", "iPadOS",
}

// GenerateFingerprint creates a random device fingerprint with consistent OS/model pairing.
func GenerateFingerprint() DeviceFingerprint {
	appVer := appVersions[rand.IntN(len(appVersions))]
	osVer := osVersions[rand.IntN(len(osVersions))]
	device := deviceModels[rand.IntN(len(deviceModels))]
	os := oss[rand.IntN(len(oss))]

	if os == "iOS" && !containsAny(device, []string{"iPhone", "iPad"}) {
		device = deviceModels[rand.IntN(3)] // picks iPhone
	}
	if os == "Android" && containsAny(device, []string{"iPhone", "iPad"}) {
		device = deviceModels[3+rand.IntN(len(deviceModels)-3)]
	}

	return DeviceFingerprint{
		AppVersion:  appVer,
		OsVersion:   osVer,
		DeviceModel: device,
		OS:          os,
	}
}

// ToClientPayload converts the fingerprint to a map for use in client payload.
func (df DeviceFingerprint) ToClientPayload() map[string]string {
	return map[string]string{
		"app_version":  df.AppVersion,
		"os_version":   df.OsVersion,
		"device_model": df.DeviceModel,
		"os":           df.OS,
	}
}

// UserAgent returns a formatted WhatsApp user agent string from the fingerprint.
func (df DeviceFingerprint) UserAgent() string {
	return "WhatsApp/" + df.AppVersion + " " + df.OS + "/" + df.OsVersion + " Device/" + df.DeviceModel
}

func mulberry32(seed int) func() float64 {
	var s = uint32(seed)
	return func() float64 {
		s += 0x6D2B79F5
		t := s ^ (s >> 15)
		t = t * (t | 1)
		t ^= t + (t ^ (t >> 7)) * (t | 61)
		return float64(^uint32(t)) / float64(^uint32(0))
	}
}

// SeededFingerprint generates a deterministic fingerprint from a seed value.
func SeededFingerprint(seed int) DeviceFingerprint {
	rng := mulberry32(seed)
	appVer := appVersions[int(rng()*float64(len(appVersions)))]
	osVer := osVersions[int(rng()*float64(len(osVersions)))]
	device := deviceModels[int(rng()*float64(len(deviceModels)))]
	os := oss[int(rng()*float64(len(oss)))]

	return DeviceFingerprint{
		AppVersion:  appVer,
		OsVersion:   osVer,
		DeviceModel: device,
		OS:          os,
	}
}

// FingerprintFromString generates a deterministic fingerprint from an arbitrary string.
func FingerprintFromString(s string) DeviceFingerprint {
	hash := 0
	for i := 0; i < len(s); i++ {
		hash = hash*31 + int(s[i])
	}
	return SeededFingerprint(hash)
}

func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if strContains(s, sub) {
			return true
		}
	}
	return false
}

func strContains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}
