package network

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type Hop struct {
	TTL     int
	Address string
	RTT     time.Duration
}

// RunTraceroute performs a traceroute using ICMP Echo Requests.
// Requires Root/Admin privileges.
func RunTraceroute(target string, maxHops int) ([]Hop, error) {
	// Resolve Target
	destAddr, err := net.ResolveIPAddr("ip4", target)
	if err != nil {
		return nil, fmt.Errorf("resolve failed: %v", err)
	}

	// Use standard net.ListenPacket for raw ICMP (Requires Root)
	c, err := net.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return nil, fmt.Errorf("listen failed (ROOT REQUIRED): %v", err)
	}
	defer c.Close()

	// Wrap with ipv4.PacketConn to set TTL
	p := ipv4.NewPacketConn(c)
	if p == nil {
		return nil, fmt.Errorf("failed to create ipv4 packet connection")
	}
	defer p.Close()

	var hops []Hop

	for ttl := 1; ttl <= maxHops; ttl++ {
		start := time.Now()

		// 1. Set TTL
		if err := p.SetTTL(ttl); err != nil {
			return hops, fmt.Errorf("failed to set TTL: %v", err)
		}

		// 2. Construct ICMP Message
		wm := icmp.Message{
			Type: ipv4.ICMPTypeEcho, Code: 0,
			Body: &icmp.Echo{
				ID: os.Getpid() & 0xffff, Seq: ttl,
				Data: []byte("NextcloudPerf"),
			},
		}
		wb, err := wm.Marshal(nil)
		if err != nil {
			continue
		}

		// 3. Send (Use p.WriteTo for raw control)
		// Note: WriteTo expects CM but we can pass nil if we set TTL on conn
		if _, err := p.WriteTo(wb, nil, destAddr); err != nil {
			continue
		}

		// 4. Wait for Reply
		c.SetReadDeadline(time.Now().Add(1 * time.Second))
		rb := make([]byte, 1500)
		n, peer, err := c.ReadFrom(rb)
		
		rtt := time.Since(start)

		if err != nil {
			// Timeout (Unreachable or no reply)
			hops = append(hops, Hop{TTL: ttl, Address: "*", RTT: 0})
			continue
		}

		// 5. Parse Reply
		// We need to parse as IPv4 ICMP
		rm, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), rb[:n])
		if err != nil {
			continue
		}

		var hopIp string
		if peer != nil {
			hopIp = peer.String()
		}

		hops = append(hops, Hop{TTL: ttl, Address: hopIp, RTT: rtt})

		// Analysis
		if rm.Type == ipv4.ICMPTypeEchoReply {
			// Reached destination
			if hopIp == destAddr.String() {
				break
			}
		}
		// TimeExceeded is expected for intermediate hops
		
		if hopIp == destAddr.String() {
			break
		}
	}
	return hops, nil
}
