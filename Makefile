# Makefile for the Calendar Solver project

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOTOOL=$(GOCMD) tool

# CLI application parameters
CLI_BINARY_NAME=calendar_solver_cli
CLI_PACKAGE=./cli

# Web application parameters
WEB_DIR=web
WASM_BINARY_NAME=$(WEB_DIR)/main.wasm
GO_WEB_PACKAGE=./$(WEB_DIR)

DOCS_DIR=docs
DOCS_ASSETS=manifest.json sw.js icon-192.png icon-512.png apple-touch-icon.png favicon.ico favicon-32.png favicon-16.png index.html wasm_exec.js

.PHONY: build_cli run_cli test_cli clean build_web run_web build_wasm web deploy

# Build the CLI application
build_cli:
	@echo "Building CLI application..."
	$(GOBUILD) -o $(CLI_BINARY_NAME) $(CLI_PACKAGE)

# Run the CLI application with optional arguments
# Example: make run_cli ARGS="--day 1 --month 1"
run_cli: build_cli
	@echo "Running CLI application..."
	./$(CLI_BINARY_NAME) $(ARGS)

# Test the CLI application
test_cli:
	@echo "Testing CLI application..."
	$(GOTEST) -v $(CLI_PACKAGE)

# Clean the project
clean:
	@echo "Cleaning up..."
	rm -f $(CLI_BINARY_NAME)
	$(GOCLEAN)

# Build the WebAssembly module
build_wasm:
	@echo "Building WebAssembly module..."
	GOOS=js GOARCH=wasm $(GOBUILD) -o $(WASM_BINARY_NAME) $(GO_WEB_PACKAGE)

# Build the web application (currently only WASM)
build_web: build_wasm

# Run the web server
run_web:
	@echo "Starting web server on http://localhost:8080"
	cd $(WEB_DIR) && $(GOCMD) run .

# Build and run the web application
web: build_web run_web

# Copy web assets to docs/ for GitHub Pages deployment
deploy: build_wasm
	@echo "Deploying to docs/..."
	@for f in $(DOCS_ASSETS); do cp $(WEB_DIR)/$$f $(DOCS_DIR)/$$f; done
	@cp $(WEB_DIR)/main.wasm $(DOCS_DIR)/main.wasm
	@echo "Done. Commit docs/ to deploy."