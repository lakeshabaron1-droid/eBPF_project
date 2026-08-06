#!/usr/bin/env bash
set -euo pipefail


GATEWAY_URL="${GATEWAY_URL:-http://127.0.0.1:8090}"
API_KEY="${API_KEY:-sk_live_gateway_prod_001}"
DURATION="${DURATION:-10}"


RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'


NC='\033[0m'

log() { echo -e "${CYAN}[load-test]${NC} $1"; }


pass() { echo -e "${GREEN}[PASS]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }

check_deps() {
    for cmd in curl ab; do

        if ! command -v "$cmd" &>/dev/null; then


            echo "Required: $cmd"
            exit 1

        fi

    done
}


baseline_test() {
    log "Scenario 1: Legitimate Traffic Baseline (10 req/s for ${DURATION}s)"

    local total=$((10 * DURATION))


    ab -n "$total" -c 10 -H "X-API-Key: ${API_KEY}" \
        "${GATEWAY_URL}/api/users" 2>/dev/null | grep -E "(Requests per second|Time taken|Failed requests|Complete requests)"


    pass "Baseline complete"
    echo ""

}



flood_test() {
    log "Scenario 2: Flood Attack Simulation (1000 req/s from single IP)"
    local total=$((1000 * DURATION))


    ab -n "$total" -c 100 -H "X-API-Key: ${API_KEY}" \
        "${GATEWAY_URL}/api/users" 2>/dev/null | grep -E "(Requests per second|Time taken|Failed requests|Complete requests)"


    pass "Flood test complete (check XDP drop counters)"
    echo ""
}


unauth_flood() {

    log "Scenario 3: Unauthenticated Flood (should all return 401)"

    ab -n 500 -c 50 \
        "${GATEWAY_URL}/api/users" 2>/dev/null | grep -E "(Requests per second|Time taken|Failed requests|Complete requests|Non-2xx)"






    pass "Unauth flood complete"

