package network

import (
	"net"
	"sort"
	"time"
)

type PingResult struct {
	Seq      int
	TimeMs   float64
	Success  bool
	ErrorMsg string
}

type DetailedPingStats struct {
	Host         string
	Count        int
	Results      []PingResult
	MinMs        float64
	MaxMs        float64
	AvgMs        float64
	SuccessCount int
	PacketLoss   float64
}

// MeasureDetailedTCPPing performs 'count' TCP connects and returns detailed stats
func MeasureDetailedTCPPing(host string, count int, timeout time.Duration) (DetailedPingStats, error) {
	stats := DetailedPingStats{
		Host:  host,
		Count: count,
	}

	var totalTime float64
	var validTimes []float64

	for i := 1; i <= count; i++ {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", host, timeout)
		duration := time.Since(start).Seconds() * 1000 // ms

		res := PingResult{
			Seq: i,
		}

		if err != nil {
			res.Success = false
			res.ErrorMsg = err.Error()
		} else {
			conn.Close()
			res.Success = true
			res.TimeMs = duration
			stats.SuccessCount++

			totalTime += duration
			validTimes = append(validTimes, duration)
		}
		stats.Results = append(stats.Results, res)
		time.Sleep(200 * time.Millisecond) // Small pause between pings
	}

	// Calculate Stats
	if stats.SuccessCount > 0 && len(validTimes) > 0 {
		stats.AvgMs = totalTime / float64(stats.SuccessCount)

		sort.Float64s(validTimes)
		stats.MinMs = validTimes[0]
		stats.MaxMs = validTimes[len(validTimes)-1]
	}

	stats.PacketLoss = (float64(count-stats.SuccessCount) / float64(count)) * 100

	return stats, nil
}

type DNSResult struct {
	Host           string
	ResolutionTime float64 // ms
	ResolvedIPs    []string
	Error          string
}

func MeasureDNS(host string) DNSResult {
	start := time.Now()
	ips, err := net.LookupIP(host)
	duration := time.Since(start).Seconds() * 1000

	res := DNSResult{
		Host:           host,
		ResolutionTime: duration,
	}

	if err != nil {
		res.Error = err.Error()
	} else {
		for _, ip := range ips {
			res.ResolvedIPs = append(res.ResolvedIPs, ip.String())
		}
	}
	return res
}
