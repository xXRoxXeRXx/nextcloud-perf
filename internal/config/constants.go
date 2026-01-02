package config

import "time"

// UI Server Configuration
const (
	LogChannelBufferSize    = 100
	ResultChannelBufferSize = 1
	DefaultServerPort       = 3000
	SSEHeartbeatInterval    = 30 * time.Second
	ClientChannelBufferSize = 10
)

// Network Tests Configuration
const (
	DefaultPingCount         = 10
	DefaultPingTimeout       = 2 * time.Second
	PingDelayBetweenTests    = 200 * time.Millisecond
	DefaultTracerouteMaxHops = 15
)

// WebDAV Configuration
const (
	DefaultChunkSize     = 25 * 1024 * 1024 // 25MB
	DefaultHTTPTimeout   = 5 * time.Minute
	MOVEOperationTimeout = 10 * time.Minute
)

// Benchmark Configuration
const (
	// Small Files Test
	SmallFileCount    = 5
	SmallFileSize     = 512 * 1024 // 512KB
	SmallFileParallel = 5

	// Medium Files Test
	MediumFileCount    = 3
	MediumFileSize     = 5 * 1024 * 1024 // 5MB
	MediumFileParallel = 1

	// Large File Test
	LargeFileSize = 256 * 1024 * 1024 // 256MB
)

// System Monitoring
const (
	CPUMonitorInterval = 2 * time.Second
	DiskBenchmarkSize  = 10 * 1024 * 1024 // 10MB
)

// Validation Limits
const (
	MaxUsernameLength = 255
	MaxPasswordLength = 1024
)
