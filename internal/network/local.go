package network

import (
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

type InterfaceInfo struct {
	Name      string
	Type      string // "Ethernet", "WiFi", "Unknown"
	IPAddress string
	LinkSpeed string // e.g., "1000 Mbps"
	IsUp      bool
}

type LocalNetworkInfo struct {
	Interfaces     []InterfaceInfo
	PrimaryIF      string
	ConnectionType string // "Ethernet", "WiFi", "Unknown"
}

func GetLocalNetworkInfo() LocalNetworkInfo {
	info := LocalNetworkInfo{}

	ifaces, err := net.Interfaces()
	if err != nil {
		return info
	}

	for _, iface := range ifaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}

		// Get first non-loopback IP
		var ipAddr string
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.IP.To4() != nil && !ipnet.IP.IsLoopback() {
					ipAddr = ipnet.IP.String()
					break
				}
			}
		}
		if ipAddr == "" {
			continue
		}

		ifInfo := InterfaceInfo{
			Name:      iface.Name,
			IPAddress: ipAddr,
			IsUp:      true,
			Type:      detectInterfaceType(iface.Name),
			LinkSpeed: getLinkSpeed(iface.Name),
		}

		info.Interfaces = append(info.Interfaces, ifInfo)

		// First active interface with IP is considered primary
		if info.PrimaryIF == "" {
			info.PrimaryIF = iface.Name
			info.ConnectionType = ifInfo.Type
		}
	}

	return info
}

func detectInterfaceType(name string) string {
	nameLower := strings.ToLower(name)

	// Common WiFi interface patterns
	if strings.HasPrefix(nameLower, "wlan") ||
		strings.HasPrefix(nameLower, "wlp") ||
		strings.HasPrefix(nameLower, "wifi") ||
		strings.Contains(nameLower, "wireless") {
		return "WiFi"
	}

	// Common Ethernet patterns
	if strings.HasPrefix(nameLower, "eth") ||
		strings.HasPrefix(nameLower, "enp") ||
		strings.HasPrefix(nameLower, "eno") ||
		strings.HasPrefix(nameLower, "ens") {
		return "Ethernet"
	}

	// macOS
	if strings.HasPrefix(nameLower, "en") {
		// en0 is usually WiFi on Mac, en1+ are ethernet
		if nameLower == "en0" {
			return "WiFi" // Usually
		}
		return "Ethernet"
	}

	return "Unknown"
}

func getLinkSpeed(ifaceName string) string {
	switch runtime.GOOS {
	case "linux":
		// Try ethtool
		out, err := exec.Command("ethtool", ifaceName).Output()
		if err == nil {
			// Parse "Speed: 1000Mb/s"
			re := regexp.MustCompile(`Speed:\s*(\d+\s*\w+)`)
			matches := re.FindStringSubmatch(string(out))
			if len(matches) > 1 {
				return matches[1]
			}
		}
		// Try /sys/class/net
		out, err = exec.Command("cat", "/sys/class/net/"+ifaceName+"/speed").Output()
		if err == nil {
			speed := strings.TrimSpace(string(out))
			if speed != "" && speed != "-1" {
				return speed + " Mbps"
			}
		}

	case "darwin":
		// macOS - try networksetup
		out, err := exec.Command("networksetup", "-getinfo", "Wi-Fi").Output()
		if err == nil && strings.Contains(string(out), "IP address") {
			return "WiFi Connected"
		}

	case "windows":
		// Windows - use PowerShell Get-NetAdapter (WMIC is deprecated on Windows 11)
		out, err := exec.Command("powershell", "-NoProfile", "-Command",
			"Get-NetAdapter | Where-Object {$_.Status -eq 'Up'} | Select-Object -First 1 -ExpandProperty LinkSpeed").Output()
		if err == nil {
			speed := strings.TrimSpace(string(out))
			if speed != "" {
				return speed
			}
		}
		// Fallback to netsh if PowerShell fails
		out, err = exec.Command("netsh", "wlan", "show", "interfaces").Output()
		if err == nil && strings.Contains(string(out), "Receive rate") {
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				if strings.Contains(line, "Receive rate") {
					parts := strings.Split(line, ":")
					if len(parts) > 1 {
						return strings.TrimSpace(parts[1])
					}
				}
			}
		}
	}

	return "Unknown"
}
