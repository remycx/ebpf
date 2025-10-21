package collector

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"os/user"
	"strconv"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
)

type ProcessEvent struct {
	Type       string `json:"type"`
	Hostname   string `json:"hostname"`
	EventType  string `json:"event_type"`
	PID        uint32 `json:"pid"`
	PPID       uint32 `json:"ppid"`
	UID        uint32 `json:"uid"`
	GID        uint32 `json:"gid"`
	Username   string `json:"username"`
	Comm       string `json:"comm"`
	Filename   string `json:"filename,omitempty"`
	Timestamp  uint64 `json:"timestamp"`
	DurationNs uint64 `json:"duration_ns,omitempty"`
	ExitCode   uint32 `json:"exit_code,omitempty"`
	CPUID      uint32 `json:"cpu_id,omitempty"`
}

type ProcessCollector struct {
	objs     *ebpf.Collection
	links    []link.Link
	execRd   *ringbuf.Reader
	exitRd   *ringbuf.Reader
	hostname string
}

func NewProcessCollector(hostname string) (*ProcessCollector, error) {
	// Remove resource limits for eBPF
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Printf("Warning: failed to remove memlock limit: %v", err)
	}

	// Load eBPF objects from compiled file
	spec, err := ebpf.LoadCollectionSpec("process_monitor.bpf.o")
	if err != nil {
		return nil, fmt.Errorf("failed to load eBPF spec: %w", err)
	}

	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to create eBPF collection: %w", err)
	}

	pc := &ProcessCollector{
		objs:     coll,
		hostname: hostname,
	}

	// Attach tracepoints
	if err := pc.attachTracepoints(); err != nil {
		pc.Close()
		return nil, err
	}

	// Setup ring buffer readers
	execMap, ok := coll.Maps["exec_events"]
	if !ok {
		pc.Close()
		return nil, fmt.Errorf("exec_events map not found")
	}

	execRd, err := ringbuf.NewReader(execMap)
	if err != nil {
		pc.Close()
		return nil, fmt.Errorf("failed to create exec ring buffer reader: %w", err)
	}
	pc.execRd = execRd

	exitMap, ok := coll.Maps["exit_events"]
	if !ok {
		pc.Close()
		return nil, fmt.Errorf("exit_events map not found")
	}

	exitRd, err := ringbuf.NewReader(exitMap)
	if err != nil {
		pc.Close()
		return nil, fmt.Errorf("failed to create exit ring buffer reader: %w", err)
	}
	pc.exitRd = exitRd

	return pc, nil
}

func (pc *ProcessCollector) attachTracepoints() error {
	tracepoints := map[string]struct {
		group string
		name  string
	}{
		"tp/sched/sched_process_exec": {"sched", "sched_process_exec"},
		"tp/sched/sched_process_exit": {"sched", "sched_process_exit"},
	}

	for progName, tp := range tracepoints {
		prog := pc.objs.Programs[progName]
		if prog == nil {
			return fmt.Errorf("program %s not found", progName)
		}

		lnk, err := link.Tracepoint(tp.group, tp.name, prog, nil)
		if err != nil {
			return fmt.Errorf("failed to attach tracepoint %s/%s: %w", tp.group, tp.name, err)
		}
		pc.links = append(pc.links, lnk)
	}

	return nil
}

func (pc *ProcessCollector) Start(eventChan chan<- interface{}) {
	// Start goroutine for exec events
	go pc.readExecEvents(eventChan)

	// Start goroutine for exit events
	go pc.readExitEvents(eventChan)
}

func (pc *ProcessCollector) readExecEvents(eventChan chan<- interface{}) {
	for {
		record, err := pc.execRd.Read()
		if err != nil {
			if err == ringbuf.ErrClosed {
				return
			}
			log.Printf("Error reading exec event: %v", err)
			continue
		}

		event, err := pc.parseExecEvent(record.RawSample)
		if err != nil {
			log.Printf("Error parsing exec event: %v", err)
			continue
		}

		eventChan <- event
	}
}

func (pc *ProcessCollector) readExitEvents(eventChan chan<- interface{}) {
	for {
		record, err := pc.exitRd.Read()
		if err != nil {
			if err == ringbuf.ErrClosed {
				return
			}
			log.Printf("Error reading exit event: %v", err)
			continue
		}

		event, err := pc.parseExitEvent(record.RawSample)
		if err != nil {
			log.Printf("Error parsing exit event: %v", err)
			continue
		}

		eventChan <- event
	}
}

func (pc *ProcessCollector) parseExecEvent(data []byte) (*ProcessEvent, error) {
	reader := bytes.NewReader(data)

	var raw struct {
		PID       uint32
		PPID      uint32
		UID       uint32
		GID       uint32
		Comm      [16]byte
		Filename  [128]byte
		Username  [32]byte
		Timestamp uint64
		CPUID     uint32
	}

	if err := binary.Read(reader, binary.LittleEndian, &raw); err != nil {
		return nil, err
	}

	username := pc.getUsername(raw.UID)

	return &ProcessEvent{
		Type:      "process",
		Hostname:  pc.hostname,
		EventType: "exec",
		PID:       raw.PID,
		PPID:      raw.PPID,
		UID:       raw.UID,
		GID:       raw.GID,
		Username:  username,
		Comm:      string(bytes.TrimRight(raw.Comm[:], "\x00")),
		Filename:  string(bytes.TrimRight(raw.Filename[:], "\x00")),
		Timestamp: raw.Timestamp,
		CPUID:     raw.CPUID,
	}, nil
}

func (pc *ProcessCollector) parseExitEvent(data []byte) (*ProcessEvent, error) {
	reader := bytes.NewReader(data)

	var raw struct {
		PID        uint32
		PPID       uint32
		UID        uint32
		GID        uint32
		Comm       [16]byte
		Timestamp  uint64
		DurationNs uint64
		ExitCode   uint32
	}

	if err := binary.Read(reader, binary.LittleEndian, &raw); err != nil {
		return nil, err
	}

	username := pc.getUsername(raw.UID)

	return &ProcessEvent{
		Type:       "process",
		Hostname:   pc.hostname,
		EventType:  "exit",
		PID:        raw.PID,
		PPID:       raw.PPID,
		UID:        raw.UID,
		GID:        raw.GID,
		Username:   username,
		Comm:       string(bytes.TrimRight(raw.Comm[:], "\x00")),
		Timestamp:  raw.Timestamp,
		DurationNs: raw.DurationNs,
		ExitCode:   raw.ExitCode,
	}, nil
}

func (pc *ProcessCollector) getUsername(uid uint32) string {
	u, err := user.LookupId(strconv.Itoa(int(uid)))
	if err != nil {
		return fmt.Sprintf("uid:%d", uid)
	}
	return u.Username
}

func (pc *ProcessCollector) Close() error {
	for _, lnk := range pc.links {
		lnk.Close()
	}
	if pc.execRd != nil {
		pc.execRd.Close()
	}
	if pc.exitRd != nil {
		pc.exitRd.Close()
	}
	if pc.objs != nil {
		pc.objs.Close()
	}
	return nil
}
