package benchmark

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"

	"nextcloud-perf/internal/webdav"
)

// MockTransport implements http.RoundTripper
type MockTransport struct {
	RoundTripFunc func(req *http.Request) *http.Response
}

func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req), nil
}

func newMockClient() *webdav.Client {
	client := webdav.NewClient("http://mock-server", "user", "pass", nil)
	client.Client.Transport = &MockTransport{
		RoundTripFunc: func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: 201,
				Body:       io.NopCloser(bytes.NewBufferString("Created")),
				Header:     make(http.Header),
			}
		},
	}
	return client
}

func TestRunSmallFiles(t *testing.T) {
	client := newMockClient()
	// Mock Transport for multiple files
	client.Client.Transport = &MockTransport{
		RoundTripFunc: func(req *http.Request) *http.Response {
			if req.Method != "PUT" {
				t.Errorf("Expected PUT, got %s", req.Method)
			}
			// Simulate some delay to test duration logic
			time.Sleep(10 * time.Millisecond)
			return &http.Response{
				StatusCode: 201,
				Body:       io.NopCloser(bytes.NewBufferString("Created")),
			}
		},
	}

	// 2 files, 1KB each, 1 parallel
	res, err := RunSmallFiles(client, "/test", "file_", 2, 1024, 1)
	if err != nil {
		t.Fatalf("RunSmallFiles failed: %v", err)
	}

	if res.Files != 2 {
		t.Errorf("Expected 2 files, got %d", res.Files)
	}
	if res.TotalSize != 2048 {
		t.Errorf("Expected 2048 bytes total, got %d", res.TotalSize)
	}
	if len(res.Errors) > 0 {
		t.Errorf("Expected no errors, got %v", res.Errors)
	}
}

func TestRunDownloadSmallFiles(t *testing.T) {
	client := newMockClient()
	// Mock Transport
	client.Client.Transport = &MockTransport{
		RoundTripFunc: func(req *http.Request) *http.Response {
			if req.Method != "GET" {
				t.Errorf("Expected GET, got %s", req.Method)
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBuffer(make([]byte, 1024))), // Return 1KB of data
			}
		},
	}

	res, err := RunDownloadSmallFiles(client, "/test", "file_", 2, 1)
	if err != nil {
		t.Fatalf("RunDownloadSmallFiles failed: %v", err)
	}

	if res.Files != 2 {
		t.Errorf("Expected 2 files, got %d", res.Files)
	}
	if res.TotalSize != 2048 {
		t.Errorf("Expected 2048 bytes download, got %d", res.TotalSize)
	}
}
