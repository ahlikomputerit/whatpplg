package antiban

import (
	"strings"
	"testing"
)

func TestGenerateFingerprint(t *testing.T) {
	fp := GenerateFingerprint()
	if fp.AppVersion == "" {
		t.Fatal("expected non-empty AppVersion")
	}
	if fp.OsVersion == "" {
		t.Fatal("expected non-empty OsVersion")
	}
	if fp.DeviceModel == "" {
		t.Fatal("expected non-empty DeviceModel")
	}
	if fp.OS == "" {
		t.Fatal("expected non-empty OS")
	}
}

func TestGenerateFingerprint_Consistency(t *testing.T) {
	fp := GenerateFingerprint()
	if fp.OS == "iOS" && !strings.Contains(fp.DeviceModel, "iPhone") && !strings.Contains(fp.DeviceModel, "iPad") {
		t.Fatalf("iOS fingerprint should have Apple device, got %s", fp.DeviceModel)
	}
	if fp.OS == "Android" && (strings.Contains(fp.DeviceModel, "iPhone") || strings.Contains(fp.DeviceModel, "iPad")) {
		t.Fatalf("Android fingerprint should not have Apple device, got %s", fp.DeviceModel)
	}
}

func TestDeviceFingerprint_ToClientPayload(t *testing.T) {
	fp := DeviceFingerprint{
		AppVersion:  "2.25.1",
		OsVersion:   "17.0",
		DeviceModel: "iPhone14,3",
		OS:          "iOS",
	}
	payload := fp.ToClientPayload()
	if payload["app_version"] != "2.25.1" {
		t.Fatalf("expected app_version 2.25.1, got %s", payload["app_version"])
	}
	if payload["os_version"] != "17.0" {
		t.Fatalf("expected os_version 17.0, got %s", payload["os_version"])
	}
	if payload["device_model"] != "iPhone14,3" {
		t.Fatalf("expected device_model iPhone14,3, got %s", payload["device_model"])
	}
	if payload["os"] != "iOS" {
		t.Fatalf("expected os iOS, got %s", payload["os"])
	}
}

func TestDeviceFingerprint_UserAgent(t *testing.T) {
	fp := DeviceFingerprint{
		AppVersion:  "2.25.1",
		OsVersion:   "17.0",
		DeviceModel: "iPhone14,3",
		OS:          "iOS",
	}
	ua := fp.UserAgent()
	expected := "WhatsApp/2.25.1 iOS/17.0 Device/iPhone14,3"
	if ua != expected {
		t.Fatalf("expected %q, got %q", expected, ua)
	}
}

func TestSeededFingerprint(t *testing.T) {
	fp1 := SeededFingerprint(42)
	fp2 := SeededFingerprint(42)
	fp3 := SeededFingerprint(99)

	if fp1.AppVersion != fp2.AppVersion {
		t.Fatal("expected same fingerprint for same seed")
	}
	if fp1.DeviceModel != fp2.DeviceModel {
		t.Fatal("expected same device model for same seed")
	}

	if fp1.AppVersion == fp3.AppVersion && fp1.DeviceModel == fp3.DeviceModel && fp1.OsVersion == fp3.OsVersion {
		t.Log("note: different seeds may still produce same values by chance")
	}
}

func TestFingerprintFromString(t *testing.T) {
	fp1 := FingerprintFromString("user123")
	fp2 := FingerprintFromString("user123")
	fp3 := FingerprintFromString("different")

	if fp1.AppVersion != fp2.AppVersion {
		t.Fatal("expected same fingerprint for same string")
	}
	if fp1.AppVersion == fp3.AppVersion && fp1.DeviceModel == fp3.DeviceModel && fp1.OsVersion == fp3.OsVersion {
		t.Log("note: different strings may still produce same values by chance")
	}
}

func TestMulberry32(t *testing.T) {
	rng := mulberry32(42)
	v1 := rng()
	v2 := rng()
	if v1 == v2 {
		t.Fatal("expected different values from sequential calls")
	}
	if v1 < 0 || v1 >= 1 {
		t.Fatalf("expected value in [0,1), got %f", v1)
	}
}
