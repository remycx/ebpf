package storage

import (
	"encoding/json"
	"sync"
	"time"
)

type Event struct {
	Type      string                 `json:"type"`
	Hostname  string                 `json:"hostname"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

type HostInfo struct {
	Hostname      string    `json:"hostname"`
	LastSeen      time.Time `json:"last_seen"`
	EventCount    int64     `json:"event_count"`
	ProcessCount  int64     `json:"process_count"`
	ConnectionCount int64   `json:"connection_count"`
}

type MemoryStore struct {
	events         []Event
	hosts          map[string]*HostInfo
	connections    []Connection
	processes      []Process
	mu             sync.RWMutex
	maxEvents      int
	eventListeners []chan Event
}

type Connection struct {
	Hostname  string `json:"hostname"`
	PID       uint32 `json:"pid"`
	Comm      string `json:"comm"`
	SrcIP     string `json:"src_ip"`
	DstIP     string `json:"dst_ip"`
	SrcPort   uint16 `json:"src_port"`
	DstPort   uint16 `json:"dst_port"`
	State     string `json:"state"`
	Timestamp int64  `json:"timestamp"`
}

type Process struct {
	Hostname  string `json:"hostname"`
	PID       uint32 `json:"pid"`
	PPID      uint32 `json:"ppid"`
	Comm      string `json:"comm"`
	Username  string `json:"username"`
	StartTime int64  `json:"start_time"`
	State     string `json:"state"`
}

func NewMemoryStore() *Store {
	return &Store{
		memory: &MemoryStore{
			events:      make([]Event, 0, 10000),
			hosts:       make(map[string]*HostInfo),
			connections: make([]Connection, 0),
			processes:   make([]Process, 0),
			maxEvents:   10000,
			eventListeners: make([]chan Event, 0),
		},
	}
}

type Store struct {
	memory *MemoryStore
}

func (s *Store) AddEvents(data []byte) error {
	var batch struct {
		Events    []json.RawMessage `json:"events"`
		Timestamp int64             `json:"timestamp"`
	}

	if err := json.Unmarshal(data, &batch); err != nil {
		return err
	}

	s.memory.mu.Lock()
	defer s.memory.mu.Unlock()

	for _, rawEvent := range batch.Events {
		var eventData map[string]interface{}
		if err := json.Unmarshal(rawEvent, &eventData); err != nil {
			continue
		}

		hostname, _ := eventData["hostname"].(string)
		eventType, _ := eventData["type"].(string)

		event := Event{
			Type:      eventType,
			Hostname:  hostname,
			Timestamp: batch.Timestamp,
			Data:      eventData,
		}

		// Add to events list
		s.memory.events = append(s.memory.events, event)
		if len(s.memory.events) > s.memory.maxEvents {
			s.memory.events = s.memory.events[1:]
		}

		// Update host info
		if hostname != "" {
			host, exists := s.memory.hosts[hostname]
			if !exists {
				host = &HostInfo{
					Hostname: hostname,
				}
				s.memory.hosts[hostname] = host
			}
			host.LastSeen = time.Now()
			host.EventCount++

			// Process specific event types
			if eventType == "network" {
				host.ConnectionCount++
				s.processNetworkEvent(eventData)
			} else if eventType == "process" {
				host.ProcessCount++
				s.processProcessEvent(eventData)
			}
		}

		// Notify listeners
		for _, listener := range s.memory.eventListeners {
			select {
			case listener <- event:
			default:
				// Skip if channel is full
			}
		}
	}

	return nil
}

func (s *MemoryStore) processNetworkEvent(data map[string]interface{}) {
	eventType, _ := data["event_type"].(string)

	if eventType == "connect" || eventType == "accept" {
		conn := Connection{
			Hostname:  data["hostname"].(string),
			PID:       uint32(data["pid"].(float64)),
			Comm:      data["comm"].(string),
			SrcIP:     data["src_ip"].(string),
			DstIP:     data["dst_ip"].(string),
			SrcPort:   uint16(data["src_port"].(float64)),
			DstPort:   uint16(data["dst_port"].(float64)),
			State:     "active",
			Timestamp: time.Now().Unix(),
		}
		s.connections = append(s.connections, conn)

		// Keep only last 1000 connections
		if len(s.connections) > 1000 {
			s.connections = s.connections[len(s.connections)-1000:]
		}
	}
}

func (s *MemoryStore) processProcessEvent(data map[string]interface{}) {
	eventType, _ := data["event_type"].(string)

	if eventType == "exec" {
		proc := Process{
			Hostname:  data["hostname"].(string),
			PID:       uint32(data["pid"].(float64)),
			PPID:      uint32(data["ppid"].(float64)),
			Comm:      data["comm"].(string),
			Username:  data["username"].(string),
			StartTime: time.Now().Unix(),
			State:     "running",
		}
		s.processes = append(s.processes, proc)

		// Keep only last 1000 processes
		if len(s.processes) > 1000 {
			s.processes = s.processes[len(s.processes)-1000:]
		}
	} else if eventType == "exit" {
		pid := uint32(data["pid"].(float64))
		hostname := data["hostname"].(string)

		for i := range s.processes {
			if s.processes[i].PID == pid && s.processes[i].Hostname == hostname {
				s.processes[i].State = "exited"
				break
			}
		}
	}
}

func (s *Store) GetHosts() []HostInfo {
	s.memory.mu.RLock()
	defer s.memory.mu.RUnlock()

	hosts := make([]HostInfo, 0, len(s.memory.hosts))
	for _, host := range s.memory.hosts {
		hosts = append(hosts, *host)
	}
	return hosts
}

func (s *Store) GetHostEvents(hostname string, limit int) []Event {
	s.memory.mu.RLock()
	defer s.memory.mu.RUnlock()

	events := make([]Event, 0)
	count := 0

	for i := len(s.memory.events) - 1; i >= 0 && count < limit; i-- {
		if s.memory.events[i].Hostname == hostname {
			events = append(events, s.memory.events[i])
			count++
		}
	}

	return events
}

func (s *Store) GetRecentEvents(limit int) []Event {
	s.memory.mu.RLock()
	defer s.memory.mu.RUnlock()

	start := len(s.memory.events) - limit
	if start < 0 {
		start = 0
	}

	events := make([]Event, len(s.memory.events)-start)
	copy(events, s.memory.events[start:])
	return events
}

func (s *Store) GetActiveConnections() []Connection {
	s.memory.mu.RLock()
	defer s.memory.mu.RUnlock()

	conns := make([]Connection, len(s.memory.connections))
	copy(conns, s.memory.connections)
	return conns
}

func (s *Store) GetActiveProcesses() []Process {
	s.memory.mu.RLock()
	defer s.memory.mu.RUnlock()

	// Return only running processes
	procs := make([]Process, 0)
	for _, proc := range s.memory.processes {
		if proc.State == "running" {
			procs = append(procs, proc)
		}
	}
	return procs
}

func (s *Store) Subscribe() <-chan Event {
	s.memory.mu.Lock()
	defer s.memory.mu.Unlock()

	ch := make(chan Event, 100)
	s.memory.eventListeners = append(s.memory.eventListeners, ch)
	return ch
}

func (s *Store) Unsubscribe(ch <-chan Event) {
	s.memory.mu.Lock()
	defer s.memory.mu.Unlock()

	for i, listener := range s.memory.eventListeners {
		if listener == ch {
			s.memory.eventListeners = append(s.memory.eventListeners[:i], s.memory.eventListeners[i+1:]...)
			close(listener)
			break
		}
	}
}

func (s *Store) GetStats() map[string]interface{} {
	s.memory.mu.RLock()
	defer s.memory.mu.RUnlock()

	return map[string]interface{}{
		"total_events":      len(s.memory.events),
		"total_hosts":       len(s.memory.hosts),
		"active_connections": len(s.memory.connections),
		"active_processes":  len(s.memory.processes),
	}
}
