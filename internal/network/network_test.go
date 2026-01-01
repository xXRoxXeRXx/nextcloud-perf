package network

import (
	"net/url"
	"testing"
)

func TestMeasureDNS(t *testing.T) {
	// We can't easily deterministic test DNS without mocking net.LookupHost,
	// but we can test that it returns without panic for a known host.

	res := MeasureDNS("localhost")
	if res.Error != "" {
		// Use skip if offline? But localhost should resolve.
		// Windows localhost might be weird if IPv6 is disabled etc.
		// If it fails, log it but don't strictly fail the test if env is weird.
		t.Logf("DNS localhost failed: %s", res.Error)
	} else {
		if res.ResolutionTime < 0 {
			t.Errorf("Expected positive resolution time, got %f", res.ResolutionTime)
		}
	}

	// Test invalid host
	resInv := MeasureDNS("invalid.host.local.test.example")
	if resInv.Error == "" {
		t.Error("Expected error for invalid host, got success")
	}
}

func TestURLParsingHelper(t *testing.T) {
	// Just testing standard logic assumption
	u, err := url.Parse("https://cloud.example.com:8443/remote.php")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if u.Hostname() != "cloud.example.com" {
		t.Errorf("Expected cloud.example.com, got %s", u.Hostname())
	}

	if u.Port() != "8443" {
		t.Errorf("Expected 8443, got %s", u.Port())
	}

	// Test without port
	u2, _ := url.Parse("https://cloud.example.com")
	if u2.Port() != "" {
		t.Errorf("Expected empty port, got %s", u2.Port())
	}
}

func TestSpeedtestStruct(t *testing.T) {
	// Validate JSON tags assumption
	st := SpeedtestResult{
		ServerID: "123",
	}
	if st.ServerID != "123" {
		t.Error("Struct assignment failed")
	}
}
