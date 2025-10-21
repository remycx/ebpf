package collector

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"syscall"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
)

type NetworkEvent struct {
	Type      string `json:"type"`
	Hostname  string `json:"hostname"`
	PID       uint32 `json:"pid"`
	UID       uint32 `json:"uid"`
	GID       uint32 `json:"gid"`
	Comm      string `json:"comm"`
	EventType string `json:"event_type"`
	SrcIP     string `json:"src_ip"`
	DstIP     string `json:"dst_ip"`
	SrcPort   uint16 `json:"src_port"`
	DstPort   uint16 `json:"dst_port"`
	Timestamp uint64 `json:"timestamp"`
}

type NetworkCollector struct {
	objs     *ebpf.Collection
	links    []link.Link
	reader   *ringbuf.Reader
	hostname string
}

func NewNetworkCollector(hostname string) (*NetworkCollector, error) {
	// Remove resource limits for eBPF
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Printf("Warning: failed to remove memlock limit: %v", err)
	}

	// Load eBPF objects from compiled file
	spec, err := ebpf.LoadCollectionSpec("network_monitor.bpf.o")
	if err != nil {
		return nil, fmt.Errorf("failed to load eBPF spec: %w", err)
	}

	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to create eBPF collection: %w", err)
	}

	nc := &NetworkCollector{
		objs:     coll,
		hostname: hostname,
	}

	// Attach kprobes
	if err := nc.attachProbes(); err != nil {
		nc.Close()
		return nil, err
	}

	// Setup ring buffer reader
	rb, ok := coll.Maps["conn_events"]
	if !ok {
		nc.Close()
		return nil, fmt.Errorf("conn_events map not found")
	}

	reader, err := ringbuf.NewReader(rb)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create ring buffer reader: %w", err)
	}
	nc.reader = reader

	return nc, nil
}

func (nc *NetworkCollector) attachProbes() error {
	probes := map[string]string{
		"kprobe/tcp_v4_connect":   "tcp_v4_connect",
		"kprobe/tcp_v6_connect":   "tcp_v6_connect",
		"kprobe/inet_csk_accept":  "inet_csk_accept",
		"kprobe/tcp_close":        "tcp_close",
	}

	for progName, funcName := range probes {
		prog := nc.objs.Programs[progName]
		if prog == nil {
			return fmt.Errorf("program %s not found", progName)
		}

		lnk, err := link.Kprobe(funcName, prog, nil)
		if err != nil {
			return fmt.Errorf("failed to attach kprobe %s: %w", funcName, err)
		}
		nc.links = append(nc.links, lnk)
	}

	return nil
}

func (nc *NetworkCollector) Start(eventChan chan<- interface{}) {
	for {
		record, err := nc.reader.Read()
		if err != nil {
			if err == ringbuf.ErrClosed {
				return
			}
			log.Printf("Error reading from ring buffer: %v", err)
			continue
		}

		event, err := nc.parseEvent(record.RawSample)
		if err != nil {
			log.Printf("Error parsing event: %v", err)
			continue
		}

		eventChan <- event
	}
}

func (nc *NetworkCollector) parseEvent(data []byte) (*NetworkEvent, error) {
	reader := bytes.NewReader(data)

	var raw struct {
		PID       uint32
		UID       uint32
		GID       uint32
		Comm      [16]byte
		Family    uint16
		SPort     uint16
		DPort     uint16
		SAddrV4   uint32
		DAddrV4   uint32
		SAddrV6   [16]byte
		DAddrV6   [16]byte
		Timestamp uint64
		EventType uint8
	}

	if err := binary.Read(reader, binary.LittleEndian, &raw); err != nil {
		return nil, err
	}

	event := &NetworkEvent{
		Type:      "network",
		Hostname:  nc.hostname,
		PID:       raw.PID,
		UID:       raw.UID,
		GID:       raw.GID,
		Comm:      string(bytes.TrimRight(raw.Comm[:], "\x00")),
		SrcPort:   raw.SPort,
		DstPort:   raw.DPort,
		Timestamp: raw.Timestamp,
	}

	switch raw.EventType {
	case 0:
		event.EventType = "connect"
	case 1:
		event.EventType = "accept"
	case 2:
		event.EventType = "close"
	default:
		event.EventType = "unknown"
	}

	if raw.Family == syscall.AF_INET {
		event.SrcIP = intToIP(raw.SAddrV4).String()
		event.DstIP = intToIP(raw.DAddrV4).String()
	} else if raw.Family == syscall.AF_INET6 {
		event.SrcIP = net.IP(raw.SAddrV6[:]).String()
		event.DstIP = net.IP(raw.DAddrV6[:]).String()
	}

	return event, nil
}

func (nc *NetworkCollector) Close() error {
	for _, lnk := range nc.links {
		lnk.Close()
	}
	if nc.reader != nil {
		nc.reader.Close()
	}
	if nc.objs != nil {
		nc.objs.Close()
	}
	return nil
}

func intToIP(ip uint32) net.IP {
	return net.IPv4(byte(ip), byte(ip>>8), byte(ip>>16), byte(ip>>24))
}
