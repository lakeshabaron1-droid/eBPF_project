# eBPF Zero-Trust Gateway

A high-performance API gateway that combines Linux eBPF packet processing with a Go reverse proxy and real-time Next.js dashboard. Packet filtering, rate limiting, and blocklist enforcement happen at the kernel level via XDP, while the userspace proxy handles dynamic routing, authentication, and authorization.



```
                          Architecture Overview

  Clients                eBPF (Kernel)                  Go Gateway (Userspace)
 ---------     ----------------------------------     -------------------------
                                                      

  HTTP  ------>  [ XDP Program ]                       [ Reverse Proxy ]
  HTTPS         |  Packet Parse (ETH/IP/TCP/UDP) |     |  TLS Termination   |
                |  Blocklist Map Lookup          |     |  Dynamic Routing   |
                |  Rate Limit (LRU + sliding)    | --> |  Connection Pool   |
                |  XDP_DROP / XDP_PASS           |     |  Health Checking   |
                |  Ring Buffer (drop events)     |     +--------------------+
                |  Packet Counters (per-CPU)     |              |
                +--------------------------------+              v
                                                      [ Auth Layer ]
                [ TC Program ]                        |  API Key Validator |
                |  Port Traffic Metrics          |     |  JWT (HS256/RS256) |
                |  Protocol Stats (TCP/UDP/ICMP) |     |  Zero-Trust RBAC   |

                |  TCP Flag Counters             |     +--------------------+
                +--------------------------------+              |
                         |                                      v
                         |                            [ Metrics Pipeline ]

                         +------------------------->  |  eBPF Map Poller   |
                                                      |  Ring Buf Consumer |
                                                      |  Time Aggregator   |
                                                      |  SSE Broadcaster   |
                                                      +--------------------+
                                                                |
                                                                v
                                                      [ Next.js Dashboard ]
                                                      |  Real-time Charts  |
                                                      |  Event Log         |

                                                      |  Control Panel     |
                                                      +--------------------+
```

## Features

**Kernel-Level Packet Processing (eBPF/XDP)**
- XDP firewall with IP blocklist enforcement at line rate
- Per-IP sliding window rate limiting using LRU hash maps

- Packet counters (per-CPU arrays) for passed/dropped traffic
- Ring buffer for streaming drop events to userspace
- TC ingress program for port traffic, protocol distribution, and TCP flag metrics

**Go Reverse Proxy**
- Path-prefix routing with longest-match-wins resolution
- Upstream connection pooling via custom http.Transport
- Active health checking with configurable thresholds



- Passive health tracking (5xx circuit breaker)
- Round-robin backend selection
- Hot-reload of routes via SIGHUP
- Graceful shutdown with in-flight request draining

**Authentication & Authorization**


- API key validation (X-API-Key header or Authorization header)
- JWT validation (HS256 and RS256) with JWKS endpoint caching
- Per-route scope-based RBAC (deny-by-default)
- Identity propagation headers (X-User-ID, X-User-Scopes, X-Auth-Method)
- Structured JSON audit logging for all auth decisions


**Middleware Pipeline**
- UUID X-Request-ID injection and propagation
- Structured JSON request/response logging

- CORS preflight handling
- Response timing capture


**Real-Time Dashboard**
- Server-Sent Events (SSE) streaming from Go backend

- Animated metric cards with sparkline SVGs
- Traffic line chart (passed vs dropped packets/sec, 60-point rolling window)
- Protocol distribution doughnut chart
- Top-10 blocked IPs horizontal bar chart
- Auto-scrolling drop event log with row flash animations
- Interactive control panel (block/unblock IP, rate limit tuning, route display)

- Glassmorphism dark-mode design system




**REST Control API**
- POST /api/block -- add IP to eBPF blocklist
- DELETE /api/block/{ip} -- remove IP from blocklist
- PUT /api/config/ratelimit -- update rate limit parameters
- GET /api/routes -- list configured routes

- GET /api/blocked -- list blocked IPs
- GET /events -- SSE metrics stream

## Prerequisites

- Linux kernel 5.15+ (tested on Kali 6.19)
- Go 1.21+
- Clang/LLVM 14+

- libbpf headers (`libbpf-dev`)

- Node.js 18+ and npm (for dashboard)
- Apache Bench (`apache2-utils`) for load testing
- Root privileges for eBPF program loading

```bash
sudo apt install -y clang llvm libbpf-dev linux-headers-$(uname -r) \
    gcc-multilib apache2-utils
```


## Quick Start


**1. Clone and build**

```bash
git clone https://github.com/lakeshabaron1-droid/eBPF_project.git
cd eBPF_project

make generate
make build
make mock-services

```

**2. Start mock backends**

```bash
./bin/service-a &
./bin/service-b &


```

**3. Start the gateway**

```bash
sudo ./bin/ebpf-gateway -config configs/gateway.yaml
```

**4. Start the dashboard**

```bash
cd dashboard
npm install
npm run dev
```

Open http://localhost:3000 in your browser.


**5. Test requests**

```bash
curl http://127.0.0.1:8090/health


curl -H "X-API-Key: sk_live_gateway_prod_001" http://127.0.0.1:8090/api/users

curl -H "X-API-Key: sk_live_gateway_admin_001" http://127.0.0.1:8090/api/admin



curl http://127.0.0.1:8081/api/routes
```

**6. Run the full demo**

```bash
./scripts/demo.sh
```

## Configuration

The gateway is configured via `configs/gateway.yaml`:

```yaml
listen:
  address: ":8090"

ebpf:
  interface: "eth0"

  xdp_mode: "generic"           # generic | native | offload
  rate_limit:
    threshold: 100              # max requests per window
    window_ms: 1000             # sliding window size in ms

auth:
  mode: "both"                  # apikey | jwt | both
  api_keys:
    - key: "sk_live_..."
      name: "frontend-app"
      scopes: ["read", "write"]
  jwt:
    algorithm: "HS256"          # HS256 | RS256
    secret: "your-secret"
    issuer: "ebpf-gateway"
    jwks_cache_ttl: 3600

routes:
  - path: "/api/users"
    upstream: "http://localhost:9001"
    auth_required: true
    required_scopes: ["read"]

    timeout_ms: 5000

dashboard:
  enabled: true
  api_address: ":8081"
  cors_origin: "http://localhost:3000"
```

## Project Structure

```
.
|-- bpf/
|   |-- headers/                  BPF helper headers
|   |-- xdp_firewall.c            XDP packet filter + rate limiter
|   |-- tc_metrics.c              TC ingress metrics collector
|-- cmd/
|   |-- gateway/main.go           Gateway entrypoint
|-- configs/
|   |-- gateway.yaml              Runtime configuration
|-- dashboard/
|   |-- src/app/                  Next.js pages and global CSS
|   |-- src/components/           React components
|   |-- src/hooks/                Custom hooks (SSE, API)


|-- internal/
|   |-- auth/                     API key, JWT, zero-trust RBAC

|   |-- config/                   YAML config parser
|   |-- ebpf/                     eBPF loader, map wrappers, bpf2go
|   |-- metrics/                  Collector, aggregator, SSE broadcaster
|   |-- proxy/                    Reverse proxy, router, middleware
|   |-- ratelimit/                Control API handlers
|-- scripts/
|   |-- load-test.sh              Load testing scenarios
|   |-- demo.sh                   Full demo orchestration
|-- test/
|   |-- integration_test.go       Integration test suite
|   |-- mock_services/            Mock backend servers
|-- Makefile
|-- go.mod

```

## API Reference

### Proxy Endpoints (port 8090)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /health | No | Health check (proxied to backend) |
| GET | /api/users | Yes (read) | User list from service A |
| GET | /api/orders | Yes (read) | Order list from service B |
| GET | /api/admin | Yes (admin) | Admin endpoint from service A |


### Control API Endpoints (port 8081)


| Method | Path | Description |
|--------|------|-------------|
| POST | /api/block | Block an IP address `{"ip": "1.2.3.4"}` |
| DELETE | /api/block/{ip} | Unblock an IP address |

| PUT | /api/config/ratelimit | Update rate limit `{"threshold": 100, "window_ms": 1000}` |
| GET | /api/routes | List all configured routes |
| GET | /api/blocked | List all blocked IPs |

| GET | /events | SSE stream of real-time metrics |

## Development

**Generate eBPF bindings**
