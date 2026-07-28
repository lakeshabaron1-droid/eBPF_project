package ebpf

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang XdpFirewall ../../bpf/xdp_firewall.c -- -I../../bpf/headers -I/usr/include/x86_64-linux-gnu
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang TcMetrics ../../bpf/tc_metrics.c -- -I../../bpf/headers -I/usr/include/x86_64-linux-gnu
