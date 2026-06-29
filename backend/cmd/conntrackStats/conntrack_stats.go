//go:build linux

package main

import (
	"fmt"
	"log"
	"net"

	"github.com/ti-mo/conntrack"
)

type HostInfo struct {
	IP            net.IP
	UploadTotal   int64
	DownloadTotal int64
}

func main() {
	// Open a Conntrack connection.
	c, err := conntrack.Dial(nil)
	if err != nil {
		log.Fatal(err)
	}

	defer c.Close()
	flows, err := c.Dump(nil)
	if err != nil {
		log.Fatal(err)
	}
	hosts := dumpLANHosts(c, flows)
	for _, h := range hosts {
		fmt.Println("h=", h)
	}
}

func dumpLANHosts(c *conntrack.Conn, flows []conntrack.Flow) map[string]*HostInfo {
	hosts := make(map[string]*HostInfo)

	for _, flow := range flows {
		// Get source and destination IPs
		srcIP := flow.TupleOrig.IP.SourceAddress
		dstIP := flow.TupleOrig.IP.DestinationAddress

		// Skip broadcast IPs (like 255.255.255.255)
		if srcIP.IsUnspecified() || dstIP.IsUnspecified() ||
			srcIP.IsMulticast() || dstIP.IsMulticast() {
			continue
		}

		// Get upstream (outbound) and downstream (inbound) traffic
		upstream := flow.CountersOrig.Bytes
		downstream := flow.CountersReply.Bytes

		// Update stats for source IP
		if srcIP.IsValid() && !srcIP.IsUnspecified() && !srcIP.IsMulticast() {
			ip := srcIP.String()
			if _, exists := hosts[ip]; !exists {
				hosts[ip] = &HostInfo{
					IP:            srcIP.AsSlice(),
					UploadTotal:   0,
					DownloadTotal: 0,
				}
			}
			hosts[ip].UploadTotal += int64(upstream)
			hosts[ip].DownloadTotal += int64(downstream)
		}

		// Update stats for destination IP
		if dstIP.IsValid() && !dstIP.IsUnspecified() && !dstIP.IsMulticast() {
			ip := dstIP.String()
			if _, exists := hosts[ip]; !exists {
				hosts[ip] = &HostInfo{
					IP:            dstIP.AsSlice(),
					UploadTotal:   0,
					DownloadTotal: 0,
				}
			}
			hosts[ip].UploadTotal += int64(downstream) // Downstream for dest is upstream for src
			hosts[ip].DownloadTotal += int64(upstream) // Upstream for dest is downstream for src
		}
	}

	return hosts
}
