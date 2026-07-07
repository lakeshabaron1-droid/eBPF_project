# eBPF Zero-Trust Reverse Proxy & API Gateway
# Build System
# ============================================================

BINARY       := ebpf-gateway
BIN_DIR      := bin
CMD_DIR      := cmd/gateway
BPF_DIR      := bpf
HEADERS_DIR  := $(BPF_DIR)/headers
DASHBOARD_DIR := dashboard

# Tools
GO           := go
CLANG        := clang
BPFTOOL      := bpftool
NPM          := npm

# Build flags
GO_BUILD_FLAGS := -ldflags="-s -w"
BPF2GO_CFLAGS := -O2 -g -Wall -Werror

# Detect architecture
ARCH := $(shell uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')

# ============================================================
# Targets
# ============================================================

.PHONY: all generate build run clean test lint \
        vmlinux dashboard dashboard-dev help

## help: Show this help message
help:
	@echo "eBPF Zero-Trust Reverse Proxy & API Gateway"
	@echo "============================================"
	@echo ""
	@echo "Targets:"
	@echo "  make generate       Generate eBPF Go bindings via bpf2go"
	@echo "  make build          Build the gateway binary"
	@echo "  make run            Build and run the gateway (requires sudo)"
	@echo "  make test           Run all Go tests"
	@echo "  make clean          Remove build artifacts"
	@echo "  make vmlinux        Generate vmlinux.h from running kernel"
	@echo "  make dashboard      Install dashboard dependencies"
	@echo "  make dashboard-dev  Run dashboard dev server"
	@echo "  make mock-services  Build mock backend services"
	@echo "  make all            Generate + Build + Dashboard"
	@echo ""

## all: Full build pipeline
all: generate build dashboard

## vmlinux: Generate vmlinux.h from running kernel's BTF data
vmlinux:
	@echo ">>> Generating vmlinux.h from /sys/kernel/btf/vmlinux..."
	$(BPFTOOL) btf dump file /sys/kernel/btf/vmlinux format c > $(HEADERS_DIR)/vmlinux.h
	@echo ">>> vmlinux.h generated ($(shell wc -l < $(HEADERS_DIR)/vmlinux.h) lines)"

## generate: Compile eBPF C programs and generate Go bindings
generate:
	@echo ">>> Generating eBPF Go bindings..."
	export BPF2GO_CFLAGS="$(BPF2GO_CFLAGS)" && \
	$(GO) generate ./internal/ebpf/...
	@echo ">>> eBPF generation complete"

## build: Build the gateway binary
build:
	@echo ">>> Building $(BINARY)..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY) ./$(CMD_DIR)
	@echo ">>> Built: $(BIN_DIR)/$(BINARY)"

## run: Build and run the gateway (requires sudo)
run: build
	@echo ">>> Starting gateway (requires root for eBPF)..."
	sudo $(BIN_DIR)/$(BINARY) -config configs/gateway.yaml

## test: Run all Go tests
test:
	@echo ">>> Running tests..."
	$(GO) test -v -race -count=1 ./internal/...
	@echo ">>> Tests complete"

## test-integration: Run integration tests (requires sudo for eBPF)
test-integration:
	@echo ">>> Running integration tests..."
	sudo $(GO) test -v -race -count=1 ./test/...
	@echo ">>> Integration tests complete"

## mock-services: Build mock backend services
mock-services:
	@echo ">>> Building mock services..."
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/service-a ./test/mock_services/service_a
	$(GO) build -o $(BIN_DIR)/service-b ./test/mock_services/service_b
	@echo ">>> Mock services built"

## dashboard: Install Next.js dashboard dependencies
dashboard:
	@echo ">>> Installing dashboard dependencies..."
	cd $(DASHBOARD_DIR) && $(NPM) install
	@echo ">>> Dashboard dependencies installed"

## dashboard-dev: Run the Next.js dashboard in dev mode
dashboard-dev:
	@echo ">>> Starting dashboard dev server..."
	cd $(DASHBOARD_DIR) && $(NPM) run dev

## dashboard-build: Build the dashboard for production
dashboard-build:
	@echo ">>> Building dashboard for production..."
	cd $(DASHBOARD_DIR) && $(NPM) run build

## clean: Remove all build artifacts and generated files
clean:
	@echo ">>> Cleaning build artifacts..."
	rm -rf $(BIN_DIR)
	rm -f internal/ebpf/*_bpfel.go internal/ebpf/*_bpfeb.go internal/ebpf/*.o
	rm -rf $(DASHBOARD_DIR)/.next $(DASHBOARD_DIR)/out
	@echo ">>> Clean complete"

## lint: Run Go linter
lint:
	@echo ">>> Linting Go code..."
	$(GO) vet ./...
	@echo ">>> Lint complete"

## deps: Download Go dependencies
deps:
	@echo ">>> Downloading Go dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo ">>> Dependencies downloaded"

## setup: Run VM setup script to install all system dependencies
setup:
	@echo ">>> Running VM setup..."
	chmod +x scripts/setup-vm.sh
	sudo scripts/setup-vm.sh
	@echo ">>> Setup complete"
