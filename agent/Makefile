# Makefile for Sentinel Agent

# Variables
CLANG ?= clang
GO ?= go
ARCH := $(shell uname -m | sed 's/x86_64/x86/' | sed 's/aarch64/arm64/')
INCLUDES := -I/usr/include/bpf -I/usr/include
BPF_CFLAGS := -g -O2 -target bpf -D__TARGET_ARCH_$(ARCH)

EBPF_SRC := ebpf
SRC_DIR := src
BUILD_DIR := build

EBPF_OBJS := $(BUILD_DIR)/network_monitor.bpf.o $(BUILD_DIR)/process_monitor.bpf.o
AGENT_BIN := $(BUILD_DIR)/sentinel-agent

.PHONY: all clean ebpf agent install

all: ebpf agent

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

# Build eBPF programs
ebpf: $(BUILD_DIR) $(EBPF_OBJS)

$(BUILD_DIR)/%.bpf.o: $(EBPF_SRC)/%.bpf.c
	$(CLANG) $(BPF_CFLAGS) $(INCLUDES) -c $< -o $@
	@echo "Built $@"

# Build agent
agent: ebpf
	cd $(SRC_DIR) && $(GO) build -o ../$(AGENT_BIN) .
	@echo "Built $(AGENT_BIN)"

# Install agent (requires root)
install: all
	install -d /etc/sentinel
	install -m 644 agent.yaml /etc/sentinel/
	install -m 755 $(AGENT_BIN) /usr/local/bin/sentinel-agent
	install -m 644 $(EBPF_OBJS) /etc/sentinel/
	@echo "Installed Sentinel Agent"

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	cd $(SRC_DIR) && $(GO) clean

# Run agent (requires root)
run: all
	@echo "Running Sentinel Agent..."
	sudo ./$(AGENT_BIN) -config agent.yaml

# Development mode - rebuild and run on changes
dev: all
	@echo "Starting development mode..."
	sudo ./$(AGENT_BIN) -config agent.yaml
