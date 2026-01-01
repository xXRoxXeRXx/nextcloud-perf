package webdav

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL  string
	Username string
	Password string
	Client   *http.Client
	LogFunc  func(string)
}

type CapabilitiesResponse struct {
	Ocs struct {
		Data struct {
			Version struct {
				Major   int    `json:"major"`
				Minor   int    `json:"minor"`
				Micro   int    `json:"micro"`
				String  string `json:"string"`
				Edition string `json:"edition"`
			} `json:"version"`
			Capabilities struct {
				Files struct {
					BigFileChunking bool `json:"bigfilechunking"`
				} `json:"files"`
				Core struct {
					PollInterval int `json:"pollinterval"`
				} `json:"core"`
			} `json:"capabilities"`
		} `json:"data"`
	} `json:"ocs"`
}

func NewClient(url, user, pass string, logFunc func(string)) *Client {
	if logFunc == nil {
		logFunc = func(s string) {}
	}
	return &Client{
		BaseURL:  url,
		Username: user,
		Password: pass,
		Client: &http.Client{
			Timeout: 5 * time.Minute,
			Transport: &http.Transport{
				TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
			},
		},
		LogFunc: logFunc,
	}
}

type StatusResponse struct {
	Installed      bool   `json:"installed"`
	Maintenance    bool   `json:"maintenance"`
	NeedsDbUpgrade bool   `json:"needsDbUpgrade"`
	Version        string `json:"version"`
	VersionString  string `json:"versionstring"`
	Edition        string `json:"edition"`
	ProductName    string `json:"productname"`
}

func (c *Client) GetStatus(ctx context.Context) (*StatusResponse, error) {
	endpoint := fmt.Sprintf("%s/status.php", c.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status.php returned: %s", resp.Status)
	}

	var status StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to parse status.php: %v", err)
	}
	return &status, nil
}

func (c *Client) GetCapabilities(ctx context.Context) (*CapabilitiesResponse, error) {
	// Ensure URL ends with / (or handle it properly).
	// The OCS endpoint is usually at /ocs/v1.php/cloud/capabilities
	// Assuming c.BaseURL is the root ID, e.g. https://cloud.example.com

	endpoint := fmt.Sprintf("%s/ocs/v1.php/cloud/capabilities?format=json", c.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("OCS-APIRequest", "true")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows) mirall/3.15.3 (build 20250107) (Nextcloud Performance Tool)")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var caps CapabilitiesResponse
	if err := json.Unmarshal(body, &caps); err != nil {
		return nil, fmt.Errorf("failed to parse capabilities: %w", err)
	}
	return &caps, nil
}

// UploadSimple performs a standard PUT upload
func (c *Client) UploadSimple(ctx context.Context, remotePath string, data io.Reader, size int64) (time.Duration, error) {
	// Construct full URL: BaseURL + /remote.php/dav/files/USER/ + remotePath
	// NOTE: This assumes BaseURL is the root. Ideally we detect the webroot.
	targetURL := fmt.Sprintf("%s/remote.php/dav/files/%s/%s", c.BaseURL, c.Username, remotePath)

	start := time.Now()
	c.LogFunc(fmt.Sprintf("PUT simple: %s (%d bytes)", targetURL, size))
	req, err := http.NewRequestWithContext(ctx, "PUT", targetURL, data)
	if size > 0 {
		req.ContentLength = size
	}
	if err != nil {
		return 0, err
	}
	req.SetBasicAuth(c.Username, c.Password)

	resp, err := c.Client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("upload failed: %s - %s", resp.Status, string(body))
	}

	return time.Since(start), nil
}

// Download retrieves a file and returns a ReadCloser
func (c *Client) Download(ctx context.Context, remotePath string) (io.ReadCloser, error) {
	targetURL := fmt.Sprintf("%s/remote.php/dav/files/%s/%s", c.BaseURL, c.Username, strings.TrimPrefix(remotePath, "/"))
	c.LogFunc(fmt.Sprintf("GET: %s", targetURL))

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.Username, c.Password)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, fmt.Errorf("download failed: %s", resp.Status)
	}

	return resp.Body, nil
}

// UploadChunked performs a Chunking V2 Upload
func (c *Client) UploadChunked(ctx context.Context, remotePath string, data io.Reader, totalSize int64) (time.Duration, error) {
	transferID := fmt.Sprintf("%d-%d", time.Now().Unix(), rand.Intn(100000))
	uploadFolder := fmt.Sprintf("%s/remote.php/dav/uploads/%s/%s", c.BaseURL, c.Username, transferID)

	start := time.Now()

	// 1. MKCOL
	c.LogFunc(fmt.Sprintf("MKCOL: %s", uploadFolder))
	req, err := http.NewRequestWithContext(ctx, "MKCOL", uploadFolder, nil)
	if err != nil {
		return 0, err
	}
	req.SetBasicAuth(c.Username, c.Password)
	resp, err := c.Client.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	if resp.StatusCode != 201 {
		return 0, fmt.Errorf("MKCOL failed: %s", resp.Status)
	}

	// 2. Upload Chunks (25MB default chunk size as requested)
	c.LogFunc("Uploading Chunks...")
	chunkSize := int64(25 * 1024 * 1024)
	buf := make([]byte, chunkSize)
	chunkIndex := 0

	for {
		n, err := data.Read(buf)
		if n == 0 {
			if err == io.EOF {
				break
			}
			return 0, err
		}

		// Upload Chunk
		// Filename format: 00001, 00002... (5 digits usually? Or just numbers. Nextcloud accepts numbers)
		// Standard recommendation: 0000000001 (padded). Let's use 5 padding.
		chunkName := fmt.Sprintf("%05d", chunkIndex+1)
		chunkURL := fmt.Sprintf("%s/%s", uploadFolder, chunkName)

		// Create a reader for just this chunk
		// We need to pass the running bytes.
		// NOTE: In a real reader loop, 'buf[:n]' is the data.
		// However, http.NewRequest needs a reader.
		// We can't reuse 'buf' because it might change? No, we block until upload is done.

		// PROBLEM: http.NewRequest with bytes.NewReader(buf[:n]) is fine.

		// If n < chunkSize, this is the last chunk.

		chunkReq, err := http.NewRequestWithContext(ctx, "PUT", chunkURL, bytes.NewReader(buf[:n]))
		if err != nil {
			return 0, err
		}
		chunkReq.SetBasicAuth(c.Username, c.Password)

		c.LogFunc(fmt.Sprintf("  > Uploading chunk %d...", chunkIndex+1))
		cRec, err := c.Client.Do(chunkReq)
		if err != nil {
			return 0, err
		}
		cRec.Body.Close()

		if cRec.StatusCode < 200 || cRec.StatusCode > 299 {
			return 0, fmt.Errorf("chunk upload failed: %s", cRec.Status)
		}

		chunkIndex++
	}

	// 3. MOVE to final destination
	// CRITICAL FIX: Move source must be the /.file virtual file inside the upload folder
	moveSource := uploadFolder + "/.file"

	// Destination Header MUST be absolute URI
	destHeaderVal := fmt.Sprintf("%s/remote.php/dav/files/%s/%s", c.BaseURL, c.Username, remotePath)

	c.LogFunc(fmt.Sprintf("MOVE %s -> %s", moveSource, destHeaderVal))

	moveReq, err := http.NewRequestWithContext(ctx, "MOVE", moveSource, nil)
	if err != nil {
		return 0, err
	}

	moveReq.Header.Set("Destination", destHeaderVal)
	moveReq.Header.Set("Overwrite", "T")
	moveReq.Header.Set("OC-Total-Length", fmt.Sprintf("%d", totalSize)) // Required for validation
	moveReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows) mirall/3.15.3 (build 20250107) (Nextcloud Performance Tool)")
	moveReq.SetBasicAuth(c.Username, c.Password)

	// Use a tailored client with long timeout for the MOVE operation
	moveClient := &http.Client{
		Timeout: 10 * time.Minute,
	}

	moveResp, err := moveClient.Do(moveReq)
	if err != nil {
		return 0, err
	} // If network fails completely
	defer moveResp.Body.Close()

	if moveResp.StatusCode < 200 || moveResp.StatusCode > 299 {
		// Attempt to read body for error details
		b, _ := io.ReadAll(moveResp.Body)
		// Debug Log
		return 0, fmt.Errorf("MOVE failed: %d %s - Dest: %s - Body: %s", moveResp.StatusCode, moveResp.Status, destHeaderVal, string(b))
	}

	return time.Since(start), nil
}

// CreateDirectory creates a folder (MKCOL)
func (c *Client) CreateDirectory(ctx context.Context, path string) error {
	fullURL := fmt.Sprintf("%s/remote.php/dav/files/%s/%s", c.BaseURL, c.Username, path)
	c.LogFunc(fmt.Sprintf("Creating Directory: %s", path))

	req, err := http.NewRequestWithContext(ctx, "MKCOL", fullURL, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.Username, c.Password)

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 405 Method Not Allowed usually means it already exists
	if resp.StatusCode == 201 || resp.StatusCode == 405 {
		return nil
	}
	return fmt.Errorf("failed to create directory %s: %s", path, resp.Status)
}

// Delete removes a file or directory
func (c *Client) Delete(ctx context.Context, path string) error {
	fullURL := fmt.Sprintf("%s/remote.php/dav/files/%s/%s", c.BaseURL, c.Username, path)
	c.LogFunc(fmt.Sprintf("Deleting: %s", path))

	req, err := http.NewRequestWithContext(ctx, "DELETE", fullURL, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.Username, c.Password)

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 || resp.StatusCode == 404 {
		return nil
	}
	return fmt.Errorf("failed to delete %s: %s", path, resp.Status)
}
