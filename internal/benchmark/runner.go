package benchmark

import (
	"crypto/rand"
	"fmt"
	"io"
	"sync"
	"time"

	"nextcloud-perf/internal/webdav"
)

// ZeroReader generates random data (or actually just zeros/rand for speed)
type ZeroReader struct {
	Limit     int64
	BytesRead int64
}

func (z *ZeroReader) Read(p []byte) (n int, err error) {
	if z.BytesRead >= z.Limit {
		return 0, io.EOF
	}
	remaining := z.Limit - z.BytesRead
	if int64(len(p)) > remaining {
		n = int(remaining)
	} else {
		n = len(p)
	}

	// Fill with random data to prevent compression cheating?
	// crypto/rand is slow. math/rand is faster but not thread safe without mutex.
	// For performance test, simple pattern or just fast filling is better.
	// Let's perform a mix: fill 1st byte random, rest zeros?
	// No, transparent compression (zfs/btrfs) will eat zeros.
	// We should fill with something non-compressible.
	// Fast way: use a pre-generated random buffer and cycle it.

	// Fill p with random data repeatedly
	copied := 0
	for copied < n {
		chunk := n - copied
		if chunk > len(GlobalRandomBuffer) {
			chunk = len(GlobalRandomBuffer)
		}
		copy(p[copied:], GlobalRandomBuffer[:chunk])
		copied += chunk
	}

	z.BytesRead += int64(n)
	return n, nil
}

// GlobalRandomBuffer is a 1MB buffer of random data
var GlobalRandomBuffer []byte

func init() {
	GlobalRandomBuffer = make([]byte, 1024*1024)
	rand.Read(GlobalRandomBuffer)
}

type Result struct {
	Scenario  string
	Files     int
	TotalSize int64
	Duration  time.Duration
	SpeedMBps float64
	Errors    []error
}

func RunSmallFiles(client *webdav.Client, basePath string, count int, size int64, parallel int) (*Result, error) {
	// Parameter validation
	if count <= 0 || size <= 0 || parallel <= 0 {
		return &Result{
			Scenario:  "Small Files (Parallel)",
			Files:     0,
			TotalSize: 0,
			Duration:  0,
			SpeedMBps: 0,
			Errors:    []error{fmt.Errorf("invalid parameters: count=%d, size=%d, parallel=%d", count, size, parallel)},
		}, nil
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, parallel) // Semaphore for concurrency control

	start := time.Now()
	var errs []error
	var mu sync.Mutex

	for i := 0; i < count; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			filename := fmt.Sprintf("%s/test_small_%d.bin", basePath, idx)
			reader := &ZeroReader{Limit: size}

			_, err := client.UploadSimple(filename, reader)
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	duration := time.Since(start)
	totalSize := int64(count) * size

	// Prevent division by zero
	var mbps float64
	if duration.Seconds() > 0 {
		mbps = float64(totalSize) / 1024 / 1024 / duration.Seconds()
	}

	return &Result{
		Scenario:  "Small Files (Parallel)",
		Files:     count,
		TotalSize: totalSize,
		Duration:  duration,
		SpeedMBps: mbps,
		Errors:    errs,
	}, nil
}

func RunLargeFile(client *webdav.Client, basePath string, size int64, useChunking bool) (*Result, error) {
	filename := fmt.Sprintf("%s/test_large.bin", basePath)
	reader := &ZeroReader{Limit: size}

	start := time.Now()
	var err error

	if useChunking {
		_, err = client.UploadChunked(filename, reader, size)
	} else {
		_, err = client.UploadSimple(filename, reader)
	}

	duration := time.Since(start)

	var errs []error
	if err != nil {
		errs = append(errs, err)
	}

	// Prevent division by zero
	var mbps float64
	if duration.Seconds() > 0 {
		mbps = float64(size) / 1024 / 1024 / duration.Seconds()
	}

	return &Result{
		Scenario:  "Large File",
		Files:     1,
		TotalSize: size,
		Duration:  duration,
		SpeedMBps: mbps,
		Errors:    errs,
	}, nil
}
