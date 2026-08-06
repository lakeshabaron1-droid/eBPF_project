#!/usr/bin/env bash
set -euo pipefail



SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"


PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
GATEWAY_PORT=8090
API_PORT=8081
DASHBOARD_PORT=3000
PIDS=()

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'


log() { echo -e "${CYAN}[demo]${NC} $1"; }
pass() { echo -e "${GREEN}[OK]${NC} $1"; }
fail() { echo -e "${RED}[FAIL]${NC} $1"; }

cleanup() {

    log "Cleaning up..."
    for pid in "${PIDS[@]}"; do
        kill "$pid" 2>/dev/null || true
        wait "$pid" 2>/dev/null || true
    done
    log "Done."
}


trap cleanup EXIT


check_deps() {
    log "Checking dependencies..."
    local missing=0
    for cmd in go make curl; do
        if ! command -v "$cmd" &>/dev/null; then
            fail "Missing: $cmd"
            missing=1
        fi
    done
    if [ "$missing" -eq 1 ]; then


        exit 1
    fi
    pass "All dependencies found"
}

build() {
    log "Building gateway and mock services..."
    cd "$PROJECT_DIR"
    make build
    make mock-services
    pass "Build complete"
}


start_backends() {
    log "Starting mock backends..."
    "$PROJECT_DIR/bin/service-a" &
    PIDS+=($!)
    "$PROJECT_DIR/bin/service-b" &

    PIDS+=($!)


    sleep 1

    pass "Mock backends running (service-a :9001, service-b :9002)"
}

start_gateway() {
    log "Starting eBPF gateway..."
    sudo "$PROJECT_DIR/bin/ebpf-gateway" -config "$PROJECT_DIR/configs/gateway.yaml" &
    PIDS+=($!)
    sleep 2


    if curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:${GATEWAY_PORT}/health" | grep -q "200"; then
        pass "Gateway running on :${GATEWAY_PORT}"
    else
        log "Gateway started (eBPF may require root)"
    fi
}


run_tests() {
    log "Running integration tests..."
    echo ""


    echo -e "${CYAN}--- Test: Health Check ---${NC}"
    curl -s -o /dev/null -w "HTTP %{http_code}\n" "http://127.0.0.1:${GATEWAY_PORT}/health"

    echo -e "${CYAN}--- Test: Authenticated Request ---${NC}"
    curl -s -H "X-API-Key: sk_live_gateway_prod_001" "http://127.0.0.1:${GATEWAY_PORT}/api/users" | python3 -m json.tool 2>/dev/null || true

    echo -e "${CYAN}--- Test: Unauthorized Request ---${NC}"
    curl -s -o /dev/null -w "HTTP %{http_code}\n" "http://127.0.0.1:${GATEWAY_PORT}/api/users"

    echo -e "${CYAN}--- Test: Forbidden (insufficient scopes) ---${NC}"
    curl -s -o /dev/null -w "HTTP %{http_code}\n" -H "X-API-Key: sk_live_gateway_prod_002" "http://127.0.0.1:${GATEWAY_PORT}/api/admin"

    echo -e "${CYAN}--- Test: Admin Access ---${NC}"
    curl -s -o /dev/null -w "HTTP %{http_code}\n" -H "X-API-Key: sk_live_gateway_admin_001" "http://127.0.0.1:${GATEWAY_PORT}/api/admin"


    echo -e "${CYAN}--- Test: Control API Routes ---${NC}"
    curl -s "http://127.0.0.1:${API_PORT}/api/routes" | python3 -m json.tool 2>/dev/null || true

    echo ""
    pass "All demo tests complete"
}


echo ""
