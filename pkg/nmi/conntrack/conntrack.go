package conntrack

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"k8s.io/klog/v2"
	knet "k8s.io/utils/net"
)

const (
	protoTCP = 6
)

// returns the netlink family for a given IP address
func getNetlinkFamily(ip net.IP) netlink.InetFamily {
	if knet.IsIPv4(ip) {
		return unix.AF_INET
	}
	return unix.AF_INET6
}

// Deletes conntrack entries for TCP connections which have metadata endpoint as their destination
func DeleteConntrackEntries(metadataIP, metadataPort string) error {
	dstIP := net.ParseIP(metadataIP)
	if dstIP == nil {
		return fmt.Errorf("metadata ip %s is incorrect", metadataIP)
	}
	dstPort, err := knet.ParsePort(metadataPort, false)
	if err != nil {
		return fmt.Errorf("failed to parse metadata port: %s, error: %w", metadataPort, err)
	}
	connectionFilter := &netlink.ConntrackFilter{}
	if err = connectionFilter.AddIP(netlink.ConntrackOrigDstIP, dstIP); err != nil {
		return fmt.Errorf("failed to delete conntrack entries, error: %w", err)
	}
	if err = connectionFilter.AddProtocol(protoTCP); err != nil {
		return fmt.Errorf("failed to delete conntrack entries, error: %w", err)
	}
	if err = connectionFilter.AddPort(netlink.ConntrackOrigDstPort, uint16(dstPort)); err != nil {
		return fmt.Errorf("failed to delete conntrack entries, error: %w", err)
	}
	connectionfamily := getNetlinkFamily(dstIP)
	klog.V(5).InfoS("net link family", connectionfamily, "ip", dstIP, "port", dstPort)
	_, err = netlink.ConntrackDeleteFilter(netlink.ConntrackTable, connectionfamily, connectionFilter)
	if err != nil {
		return fmt.Errorf("failed to delete conntrack entries, error: %w", err)
	}
	klog.V(5).Info("deleted conntrack entries")
	return nil
}
