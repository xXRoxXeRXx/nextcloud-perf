package system

import (
	"fmt"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type SystemInfo struct {
	Hostname        string
	OS              string
	Platform        string
	PlatformVersion string
	KernelArch      string
	Uptime          time.Duration
	CPUModel        string
	CPUCores        int
	CPUUsage        float64 // Percent
	RAMTotal        uint64
	RAMFree         uint64
	RAMUsed         uint64
	RAMUsage        float64 // Percent
}

func GetSystemInfo() (*SystemInfo, error) {
	h, err := host.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get host info: %w", err)
	}

	vm, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory info: %w", err)
	}

	cpuInfo, err := cpu.Info()
	cpuModel := "Unknown"
	if err == nil && len(cpuInfo) > 0 {
		cpuModel = cpuInfo[0].ModelName
	}

	// Calculate CPU Usage (Blocking for 1 second to measure)
	cpuPercents, _ := cpu.Percent(1*time.Second, false)
	cpuUsage := 0.0
	if len(cpuPercents) > 0 {
		cpuUsage = cpuPercents[0]
	}

	return &SystemInfo{
		Hostname:        h.Hostname,
		OS:              h.OS,
		Platform:        h.Platform,
		PlatformVersion: h.PlatformVersion,
		KernelArch:      h.KernelArch,
		Uptime:          time.Duration(h.Uptime) * time.Second,
		CPUModel:        cpuModel,
		CPUCores:        runtime.NumCPU(),
		CPUUsage:        cpuUsage,
		RAMTotal:        vm.Total,
		RAMFree:         vm.Available,
		RAMUsed:         vm.Used,
		RAMUsage:        vm.UsedPercent,
	}, nil
}

