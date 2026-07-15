package ebpf

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang XdpFirewall ../../bpf/xdp_firewall.c -- -I../../bpf/headers
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang TcMetrics ../../bpf/tc_metrics.c -- -I../../bpf/headers
