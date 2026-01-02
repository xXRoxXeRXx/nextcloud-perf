package webdav

import (
	"errors"
	"fmt"
)

// Sentinel errors for WebDAV operations
var (
	ErrMOVEFailed        = errors.New("MOVE operation failed")
	ErrChunkUploadFailed = errors.New("chunk upload failed")
	ErrMKCOLFailed       = errors.New("MKCOL operation failed")
	ErrDeleteFailed      = errors.New("DELETE operation failed")
	ErrUnauthorized      = errors.New("authentication failed")
	ErrNotFound          = errors.New("resource not found")
	ErrPUTFailed         = errors.New("PUT operation failed")
	ErrGETFailed         = errors.New("GET operation failed")
	ErrPROPFINDFailed    = errors.New("PROPFIND operation failed")
)

// NewMOVEError wraps ErrMOVEFailed with additional context
func NewMOVEError(statusCode int, body string) error {
	return fmt.Errorf("%w: HTTP %d - %s", ErrMOVEFailed, statusCode, body)
}

// NewChunkUploadError wraps ErrChunkUploadFailed with additional context
func NewChunkUploadError(chunkNum int, err error) error {
	return fmt.Errorf("%w: chunk %d - %v", ErrChunkUploadFailed, chunkNum, err)
}

// NewMKCOLError wraps ErrMKCOLFailed with additional context
func NewMKCOLError(statusCode int, path string) error {
	return fmt.Errorf("%w: HTTP %d for path %s", ErrMKCOLFailed, statusCode, path)
}

// NewDeleteError wraps ErrDeleteFailed with additional context
func NewDeleteError(statusCode int, path string) error {
	return fmt.Errorf("%w: HTTP %d for path %s", ErrDeleteFailed, statusCode, path)
}

// NewPUTError wraps ErrPUTFailed with additional context
func NewPUTError(statusCode int, path string) error {
	return fmt.Errorf("%w: HTTP %d for path %s", ErrPUTFailed, statusCode, path)
}

// NewGETError wraps ErrGETFailed with additional context
func NewGETError(statusCode int, path string) error {
	return fmt.Errorf("%w: HTTP %d for path %s", ErrGETFailed, statusCode, path)
}
