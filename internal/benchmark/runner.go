package benchmark

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"sync"
	"time"

	"nextcloud-perf/internal/webdav"
)

// ZeroReader generates random data for upload benchmarks.
// It uses a pre-generated random buffer with offset-based reading
// to avoid crypto overhead while preventing transparent compression detection.
type ZeroReader struct {
	Limit        int64 // Maximum bytes to read
	BytesRead    int64 // Total bytes read so far
	bufferOffset int   // Current offset in GlobalRandomBuffer
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

	// Copy with variable offset to prevent pattern recognition
	// This makes each ZeroReader instance produce unique data
	copied := 0
	for copied < n {
		available := len(GlobalRandomBuffer) - z.bufferOffset
		toCopy := n - copied
		if toCopy > available {
			toCopy = available
		}
		copy(p[copied:], GlobalRandomBuffer[z.bufferOffset:z.bufferOffset+toCopy])
		copied += toCopy
		// Wrap around to beginning of buffer
		z.bufferOffset = (z.bufferOffset + toCopy) % len(GlobalRandomBuffer)
	}

	z.BytesRead += int64(n)
	return n, nil
}

// GlobalRandomBuffer is a 10MB buffer of random data used by ZeroReader
// to generate non-compressible test data efficiently.
// Larger buffer size reduces pattern repetition in large uploads.
var GlobalRandomBuffer []byte

func init() {
	// Use 10MB buffer instead of 1MB for better randomness in large files
	GlobalRandomBuffer = make([]byte, 10*1024*1024)
	if _, err := rand.Read(GlobalRandomBuffer); err != nil {
		panic(fmt.Sprintf("failed to initialize random buffer: %v", err))
	}
}

// Result contains the performance metrics from a benchmark run.
type Result struct {
	Scenario  string        // Name of the benchmark scenario
	Files     int           // Number of files processed
	TotalSize int64         // Total bytes transferred
	Duration  time.Duration // Time taken for the operation
	SpeedMBps float64       // Transfer speed in MB/s
	Errors    []error       // Collection of errors encountered
}

// RunSmallFiles performs a parallel upload benchmark with multiple small files.
// It uploads 'count' files of 'size' bytes each using 'parallel' concurrent goroutines.
//
// Parameters:
//   - ctx: Context for cancellation
//   - client: WebDAV client configured for target server
//   - basePath: Remote directory path for test files
//   - filePrefix: Prefix for generated test filenames
//   - count: Number of files to upload
//   - size: Size of each file in bytes
//   - parallel: Maximum number of concurrent uploads
//
// Returns:
//   - *Result: Aggregated performance metrics including speed and duration
//   - error: Error if benchmark cannot be initialized (nil on success)
//
// Individual file upload errors are collected in Result.Errors but do not
// prevent the benchmark from completing.
func RunSmallFiles(ctx context.Context, client *webdav.Client, basePath string, filePrefix string, count int, size int64, parallel int) (*Result, error) {
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
	errChan := make(chan error, count)   // Buffered channel for errors

	start := time.Now()

	for i := 0; i < count; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			filename := fmt.Sprintf("%s/%s%d.bin", basePath, filePrefix, idx)
			reader := &ZeroReader{Limit: size}

			_, err := client.UploadSimple(ctx, filename, reader, size)
			if err == nil {
				client.LogFunc(fmt.Sprintf("DEBUG: Uploaded %d bytes to %s", size, filename))
			}
			if err != nil {
				select {
				case errChan <- err:
				default:
					// Skip if channel is full
				}
			}
		}(i)
	}
	wg.Wait()
	close(errChan)

	// Collect errors after all goroutines finished
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

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

// RunLargeFile performs a single large file upload benchmark with optional chunking.
//
// Parameters:
//   - ctx: Context for cancellation
//   - client: WebDAV client configured for target server
//   - basePath: Remote directory path for test file
//   - size: Size of the file in bytes
//   - useChunking: If true, uses chunked upload protocol (recommended for files > 50MB)
//
// Returns:
//   - *Result: Performance metrics including speed and duration
//   - error: Error if upload fails completely (nil on success)
//
// This function is optimized for large files and uses streaming to avoid
// loading the entire file into memory.
func RunLargeFile(ctx context.Context, client *webdav.Client, basePath string, size int64, useChunking bool) (*Result, error) {
	filename := fmt.Sprintf("%s/test_large.bin", basePath)
	reader := &ZeroReader{Limit: size}

	start := time.Now()
	var err error

	if useChunking {
		_, err = client.UploadChunked(ctx, filename, reader, size)
	} else {
		_, err = client.UploadSimple(ctx, filename, reader, size)
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

// DiscardReader reads from r and discards everything (like /dev/null), counting bytes
func (z *ZeroReader) ReadFrom(r io.Reader) (n int64, err error) {
	buf := make([]byte, 32*1024)
	for {
		nr, er := r.Read(buf)
		if nr > 0 {
			n += int64(nr)
		}
		if er != nil {
			if er == io.EOF {
				return n, nil
			}
			return n, er
		}
	}
}

// RunDownloadSmallFiles performs a parallel download benchmark with multiple small files.
//
// Parameters:
//   - ctx: Context for cancellation
//   - client: WebDAV client configured for target server
//   - basePath: Remote directory path containing test files
//   - filePrefix: Prefix of test filenames to download
//   - count: Number of files to download
//   - parallel: Maximum number of concurrent downloads
//
// Returns:
//   - *Result: Performance metrics including download speed and duration
//   - error: Error if benchmark cannot be initialized (nil on success)
//
// Files must exist on the server (typically created by RunSmallFiles).
// Individual file download errors are collected in Result.Errors.
func RunDownloadSmallFiles(ctx context.Context, client *webdav.Client, basePath string, filePrefix string, count int, parallel int) (*Result, error) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, parallel)
	errChan := make(chan error, count)
	bytesChan := make(chan int64, count)

	start := time.Now()

	for i := 0; i < count; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			filename := fmt.Sprintf("%s/%s%d.bin", basePath, filePrefix, idx)
			rc, err := client.Download(ctx, filename)
			if err != nil {
				select {
				case errChan <- err:
				default:
				}
				return
			}
			defer rc.Close()

			// Read and discard
			written, errCopy := io.Copy(io.Discard, rc)
			if errCopy != nil {
				client.LogFunc(fmt.Sprintf("Download stream error: %v", errCopy))
			}
			client.LogFunc(fmt.Sprintf("DEBUG: Downloaded %d bytes from %s", written, filename))
			
			bytesChan <- written
		}(i)
	}
	wg.Wait()
	close(errChan)
	close(bytesChan)

	// Collect errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	// Sum bytes
	var totalBytes int64
	for b := range bytesChan {
		totalBytes += b
	}

	duration := time.Since(start)

	var mbps float64
	if duration.Seconds() > 0 {
		mbps = float64(totalBytes) / 1024 / 1024 / duration.Seconds()
	}

	return &Result{
		Scenario:  "Small Files Download",
		Files:     count,
		TotalSize: totalBytes,
		Duration:  duration,
		SpeedMBps: mbps,
		Errors:    errs,
	}, nil
}

// RunDownloadLargeFile performs a single large file download benchmark.
//
// Parameters:
//   - ctx: Context for cancellation
//   - client: WebDAV client configured for target server
//   - basePath: Remote directory path containing the test file
//
// Returns:
//   - *Result: Performance metrics including download speed and duration
//   - error: Error if download fails
//
// The file must exist on the server (typically created by RunLargeFile).
// This function uses streaming to avoid loading the entire file into memory.
func RunDownloadLargeFile(ctx context.Context, client *webdav.Client, basePath string) (*Result, error) {
	filename := fmt.Sprintf("%s/test_large.bin", basePath)

	start := time.Now()
	rc, err := client.Download(ctx, filename)

	var totalBytes int64
	var errs []error

	if err != nil {
		errs = append(errs, err)
	} else {
		defer rc.Close()
		totalBytes, err = io.Copy(io.Discard, rc)
		if err != nil {
			errs = append(errs, err)
		}
	}

	duration := time.Since(start)

	var mbps float64
	if duration.Seconds() > 0 {
		mbps = float64(totalBytes) / 1024 / 1024 / duration.Seconds()
	}

	return &Result{
		Scenario:  "Large File Download",
		Files:     1,
		TotalSize: totalBytes,
		Duration:  duration,
		SpeedMBps: mbps,
		Errors:    errs,
	}, nil
}
