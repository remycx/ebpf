# Makefile for Sentinel Backend

GO ?= go
BUILD_DIR := build
BINARY := sentinel

.PHONY: all build clean run dev

all: build

build:
	mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(BINARY) .
	@echo "Built $(BUILD_DIR)/$(BINARY)"

clean:
	rm -rf $(BUILD_DIR)
	$(GO) clean

run: build
	./$(BUILD_DIR)/$(BINARY) -config config.yaml

dev:
	$(GO) run . -config config.yaml

install: build
	install -m 755 $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)
	install -d /etc/sentinel
	install -m 644 config.yaml /etc/sentinel/sentinel.yaml
	@echo "Installed Sentinel Server"
