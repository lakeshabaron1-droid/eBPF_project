#include <linux/bpf.h>
#include <linux/pkt_cls.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <linux/in.h>
#include "headers/bpf_helpers.h"
#include "headers/bpf_endian.h"

char __license[] SEC("license") = "Dual BSD/GPL";

struct port_stats {
    __u64 packets;
    __u64 bytes;
};

struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_HASH);
    __uint(max_entries, 65536);
    __type(key, __u16);
    __type(value, struct port_stats);
} port_metrics SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, 4);
    __type(key, __u32);
    __type(value, __u64);
} protocol_metrics SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, 3);
    __type(key, __u32);
    __type(value, __u64);
} tcp_flags_metrics SEC(".maps");

static __always_inline void record_protocol(__u8 protocol) {
    __u32 key;
    if (protocol == IPPROTO_TCP)
        key = 0;
    else if (protocol == IPPROTO_UDP)
        key = 1;
    else if (protocol == IPPROTO_ICMP)
        key = 2;
    else
        key = 3;

    __u64 *val = bpf_map_lookup_elem(&protocol_metrics, &key);
    if (val) {
        __sync_fetch_and_add(val, 1);
    }
}

static __always_inline void record_tcp_flags(struct tcphdr *tcp) {
    if (tcp->syn) {
        __u32 key = 0;
        __u64 *val = bpf_map_lookup_elem(&tcp_flags_metrics, &key);
        if (val) __sync_fetch_and_add(val, 1);
    }
    if (tcp->fin) {
        __u32 key = 1;
        __u64 *val = bpf_map_lookup_elem(&tcp_flags_metrics, &key);
        if (val) __sync_fetch_and_add(val, 1);
    }
    if (tcp->rst) {
        __u32 key = 2;
        __u64 *val = bpf_map_lookup_elem(&tcp_flags_metrics, &key);
        if (val) __sync_fetch_and_add(val, 1);
    }
}

static __always_inline void record_port_traffic(__u16 port, __u32 bytes) {
    struct port_stats *stats = bpf_map_lookup_elem(&port_metrics, &port);
    if (!stats) {
        struct port_stats new_stats;
        new_stats.packets = 1;
        new_stats.bytes = bytes;
        bpf_map_update_elem(&port_metrics, &port, &new_stats, BPF_ANY);
    } else {
        __sync_fetch_and_add(&stats->packets, 1);
        __sync_fetch_and_add(&stats->bytes, bytes);
    }
}

SEC("tc")
int tc_ingress_metrics(struct __sk_buff *skb) {
    void *data_end = (void *)(long)skb->data_end;
    void *data = (void *)(long)skb->data;
    __u32 bytes = skb->len;

    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return TC_ACT_OK;

    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return TC_ACT_OK;

    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end)
        return TC_ACT_OK;

    record_protocol(ip->protocol);

    if (ip->protocol == IPPROTO_TCP) {
        struct tcphdr *tcp = (void *)ip + (ip->ihl * 4);
        if ((void *)(tcp + 1) > data_end)
            return TC_ACT_OK;

        __u16 dst_port = bpf_ntohs(tcp->dest);
        record_port_traffic(dst_port, bytes);
        record_tcp_flags(tcp);
    } else if (ip->protocol == IPPROTO_UDP) {
        struct udphdr *udp = (void *)ip + (ip->ihl * 4);
        if ((void *)(udp + 1) > data_end)
            return TC_ACT_OK;

        __u16 dst_port = bpf_ntohs(udp->dest);
        record_port_traffic(dst_port, bytes);
    }

    return TC_ACT_OK;
}
