package network

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptrace"
	"os"
	"strings"
	"time"
)

type ExtendedNetworkInfo struct {
	TLSHandshakeMs float64
	ProxyDetected  bool
	VPNDetected    bool
	VPNType        string
	MTU            int
}

func MeasureTLSHandshake(targetURL string) (time.Duration, error) {
	var start, connect, dnsDone, tlsStart time.Duration
	var tlsHandshake time.Duration

	trace := &httptrace.ClientTrace{
		DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
			dnsDone = time.Duration(time.Now().UnixNano())
		},
		ConnectDone: func(network, addr string, err error) {
			connect = time.Duration(time.Now().UnixNano())
		},
		TLSHandshakeStart: func() {
			tlsStart = time.Duration(time.Now().UnixNano())
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			tlsHandshake = time.Since(time.Unix(0, int64(tlsStart)))
		},
	}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return 0, err
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	start = time.Duration(time.Now().UnixNano()) // dummy just for completeness
	_ = start
	_ = dnsDone
	_ = connect

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return tlsHandshake, nil
}

func GetExtendedNetworkInfo() ExtendedNetworkInfo {
	info := ExtendedNetworkInfo{}

	// 1. Proxy Detection
	if os.Getenv("HTTP_PROXY") != "" || os.Getenv("HTTPS_PROXY") != "" || os.Getenv("http_proxy") != "" || os.Getenv("https_proxy") != "" {
		info.ProxyDetected = true
	}

	// 2. VPN Detection & MTU
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range interfaces {
			// Check for typical VPN interface names
			name := strings.ToLower(iface.Name)
			if strings.Contains(name, "tun") || strings.Contains(name, "tap") ||
				strings.Contains(name, "wg") || strings.Contains(name, "wireguard") ||
				strings.Contains(name, "ppp") || strings.Contains(name, "vpn") ||
				strings.Contains(name, "tailscale") || strings.Contains(name, "zerotier") {

				// Only count if it's up
				if iface.Flags&net.FlagUp != 0 {
					info.VPNDetected = true
					info.VPNType = iface.Name
				}
			}

			// Capture the MTU of the likely primary interface
			// Typical primary interfaces are eth0, en0, Wi-Fi, Ethernet, etc.
			if (strings.HasPrefix(name, "eth") || strings.HasPrefix(name, "en") ||
				strings.HasPrefix(name, "wi-fi") || strings.HasPrefix(name, "wlan") ||
				strings.HasPrefix(name, "ethernet")) && iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
				if info.MTU == 0 || iface.MTU < info.MTU {
					info.MTU = iface.MTU
				}
			}
		}
	}

	return info
}
