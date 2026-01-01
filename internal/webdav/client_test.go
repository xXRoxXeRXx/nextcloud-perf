package webdav

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetCapabilities(t *testing.T) {
	// Mock Server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Request
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.Header.Get("OCS-APIRequest") != "true" {
			t.Error("Missing OCS-APIRequest header")
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "testuser" || pass != "testpass" {
			t.Error("Invalid Auth")
		}

		// Response
		resp := CapabilitiesResponse{}
		resp.Ocs.Data.Version.String = "25.0.0"
		resp.Ocs.Data.Capabilities.Files.BigFileChunking = true

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer ts.Close()

	// Test Client
	client := NewClient(ts.URL, "testuser", "testpass", nil)
	caps, err := client.GetCapabilities()
	if err != nil {
		t.Fatalf("Failed to get capabilities: %v", err)
	}

	if caps.Ocs.Data.Version.String != "25.0.0" {
		t.Errorf("Expected version 25.0.0, got %s", caps.Ocs.Data.Version.String)
	}
	if !caps.Ocs.Data.Capabilities.Files.BigFileChunking {
		t.Error("Expected BigFileChunking to be true")
	}
}

func TestDownload(t *testing.T) {
	expectedContent := "Hello, World! This is a test file."

	// Mock Server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		// Basic check for path structure
		if r.URL.Path != "/remote.php/dav/files/testuser/test.txt" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "text/plain")
		if _, err := w.Write([]byte(expectedContent)); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer ts.Close()

	// Test Client
	client := NewClient(ts.URL, "testuser", "testpass", nil)

	// Execute Download
	rc, err := client.Download("/test.txt")
	if err != nil {
		t.Fatalf("Failed to download: %v", err)
	}
	defer rc.Close()

	// Verify Content
	content, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("Failed to read content: %v", err)
	}

	if string(content) != expectedContent {
		t.Errorf("Expected content %q, got %q", expectedContent, string(content))
	}
}
