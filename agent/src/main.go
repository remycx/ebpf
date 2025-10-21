package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sentinel/agent/pkg/collector"
	"github.com/sentinel/agent/pkg/config"
	"github.com/sentinel/agent/pkg/sender"
)

var (
	configFile = flag.String("config", "/etc/sentinel/agent.yaml", "Path to configuration file")
	version    = "1.0.0"
)

func main() {
	flag.Parse()

	log.Printf("Sentinel Agent v%s starting...", version)

	// Load configuration
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate we're running as root (required for eBPF)
	if os.Geteuid() != 0 {
		log.Fatal("This program must be run as root (for eBPF)")
	}

	// Initialize event sender
	eventSender, err := sender.New(cfg.Server)
	if err != nil {
		log.Fatalf("Failed to initialize sender: %v", err)
	}
	defer eventSender.Close()

	// Initialize collectors
	networkCollector, err := collector.NewNetworkCollector(cfg.Agent.Hostname)
	if err != nil {
		log.Fatalf("Failed to initialize network collector: %v", err)
	}
	defer networkCollector.Close()

	processCollector, err := collector.NewProcessCollector(cfg.Agent.Hostname)
	if err != nil {
		log.Fatalf("Failed to initialize process collector: %v", err)
	}
	defer processCollector.Close()

	// Setup event channels
	networkEvents := make(chan interface{}, 1000)
	processEvents := make(chan interface{}, 1000)

	// Start collectors
	go networkCollector.Start(networkEvents)
	go processCollector.Start(processEvents)

	// Start event aggregator
	go aggregateAndSend(networkEvents, processEvents, eventSender)

	log.Printf("Agent started successfully. Monitoring system events...")

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down agent...")
}

func aggregateAndSend(networkChan, processChan <-chan interface{}, sender *sender.Sender) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	batch := make([]interface{}, 0, 100)

	for {
		select {
		case event := <-networkChan:
			batch = append(batch, event)
			if len(batch) >= 50 {
				sendBatch(sender, batch)
				batch = batch[:0]
			}

		case event := <-processChan:
			batch = append(batch, event)
			if len(batch) >= 50 {
				sendBatch(sender, batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				sendBatch(sender, batch)
				batch = batch[:0]
			}
		}
	}
}

func sendBatch(sender *sender.Sender, batch []interface{}) {
	data, err := json.Marshal(map[string]interface{}{
		"events":    batch,
		"timestamp": time.Now().Unix(),
	})
	if err != nil {
		log.Printf("Failed to marshal batch: %v", err)
		return
	}

	if err := sender.Send(data); err != nil {
		log.Printf("Failed to send batch: %v", err)
	}
}
