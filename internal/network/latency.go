package network

import (
	"net"
	"time"
)

type LatencyStats struct {
	Min         time.Duration
	Max         time.Duration
	Avg         time.Duration
	Jitter      time.Duration
	PacketLoss  float64
	Count       int
	Success     int
}

// MeasureTCPPing connects to a target (host:port) multiple times to estimate latency and jitter.
// This requires NO admin privileges.
func MeasureTCPPing(target string, count int, timeout time.Duration) (*LatencyStats, error) {
	var latencies []time.Duration
	success := 0

	for i := 0; i < count; i++ {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", target, timeout)
		duration := time.Since(start)

		if err == nil {
			conn.Close()
			latencies = append(latencies, duration)
			success++
		}
		
		// Sleep a bit between pings to mimic real behavior and not get flagged as DoS
		time.Sleep(200 * time.Millisecond)
	}

	if len(latencies) == 0 {
		return &LatencyStats{PacketLoss: 100.0, Count: count, Success: 0}, nil
	}

	// Calculate stats
	var total, minVal, maxVal time.Duration
	minVal = latencies[0]
	
	for _, l := range latencies {
		total += l
		if l < minVal {
			minVal = l
		}
		if l > maxVal {
			maxVal = l
		}
	}
	avg := total / time.Duration(len(latencies))

	// Calculate Jitter (Mean Deviation from Average)
	var jitterSum time.Duration
	for _, l := range latencies {
		diff := l - avg
		if diff < 0 {
			diff = -diff
		}
		jitterSum += diff
	}
	jitter := jitterSum / time.Duration(len(latencies))

	loss := float64(count-success) / float64(count) * 100.0

	return &LatencyStats{
		Min:        minVal,
		Max:        maxVal,
		Avg:        avg,
		Jitter:     jitter,
		PacketLoss: loss,
		Count:      count,
		Success:    success,
	}, nil
}
