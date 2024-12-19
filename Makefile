# Makefile

# Variables
APP_NAME := auction-server
BUILD_DIR := cmd/
BIN_NAME := server
CONFIG_FILE := ./configs/config.yaml

# Air target for hot reloading
.PHONY: dev
dev:
	@echo "Starting development server with Air..."
	air -c .air.toml

# Build production binary
.PHONY: build
build:
	@echo "Building production binary..."
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BIN_NAME) ./$(BUILD_DIR)
	@echo "Build completed: $(BUILD_DIR)/$(BIN_NAME)"

# Clean up binaries and temporary files
.PHONY: clean
clean:
	@echo "Cleaning up..."
	rm -rf $(BUILD_DIR)/$(BIN_NAME) tmp air.log
	@echo "Clean up completed."

# Run server binary
.PHONY: run
run:
	@echo "Running server binary..."
	$(BUILD_DIR)/$(BIN_NAME) --config=$(CONFIG_FILE)
