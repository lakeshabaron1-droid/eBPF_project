#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <linux/in.h>
#include "headers/bpf_helpers.h"
#include "headers/bpf_endian.h"

char __license[] SEC("license") = "Dual BSD/GPL";

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 100000);
    __type(key, __u32);
    __type(value, __u8);
} blocklist_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, 2);
    __type(key, __u32);
    __type(value, __u64);
} packet_counters SEC(".maps");

struct rl_config {
    __u32 threshold;
    __u32 window_ms;
};

struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, struct rl_config);
} rate_limit_config SEC(".maps");

struct rl_state {
    __u64 window_start;
    __u32 count;
};

struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, 100000);
    __type(key, __u32);
    __type(value, struct rl_state);
} rate_limit_state SEC(".maps");

struct drop_event {
    __u32 src_ip;
    __u32 dst_ip;
    __u16 src_port;
    __u16 dst_port;
    __u8 protocol;
    __u8 reason;
};

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 262144);
} drop_events SEC(".maps");

static __always_inline int parse_packet(void *data, void *data_end, struct drop_event *event) {
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return -1;

    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return -1;

    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end)
        return -1;

    event->src_ip = ip->saddr;
    event->dst_ip = ip->daddr;
    event->protocol = ip->protocol;

    if (ip->protocol == IPPROTO_TCP) {
        struct tcphdr *tcp = (void *)ip + (ip->ihl * 4);
        if ((void *)(tcp + 1) > data_end)
            return -1;
        event->src_port = bpf_ntohs(tcp->source);
        event->dst_port = bpf_ntohs(tcp->dest);
    } else if (ip->protocol == IPPROTO_UDP) {
        struct udphdr *udp = (void *)ip + (ip->ihl * 4);
        if ((void *)(udp + 1) > data_end)
            return -1;
        event->src_port = bpf_ntohs(udp->source);
        event->dst_port = bpf_ntohs(udp->dest);
    } else {
        event->src_port = 0;
        event->dst_port = 0;
    }

    return 0;
}

static __always_inline void emit_drop_event(struct drop_event *event, __u8 reason) {
    event->reason = reason;
    struct drop_event *ring_event = bpf_ringbuf_reserve(&drop_events, sizeof(struct drop_event), 0);
    if (ring_event) {
        *ring_event = *event;
        bpf_ringbuf_submit(ring_event, 0);
    }
}

static __always_inline void increment_counter(__u32 index) {
    __u64 *count = bpf_map_lookup_elem(&packet_counters, &index);
    if (count) {
        __sync_fetch_and_add(count, 1);
    }
}

SEC("xdp")
int xdp_firewall(struct xdp_md *ctx) {
    void *data_end = (void *)(long)ctx->data_end;
    void *data = (void *)(long)ctx->data;
    struct drop_event event = {0};

    if (parse_packet(data, data_end, &event) < 0) {
        increment_counter(0);
        return XDP_PASS;
    }

    __u32 src_ip = event.src_ip;
    __u8 *blocked = bpf_map_lookup_elem(&blocklist_map, &src_ip);
    if (blocked && *blocked == 1) {
        increment_counter(1);
        emit_drop_event(&event, 1);
        return XDP_DROP;
    }

    __u32 config_key = 0;
    struct rl_config *config = bpf_map_lookup_elem(&rate_limit_config, &config_key);
    if (config && config->threshold > 0) {
        __u64 now = bpf_ktime_get_ns();
        struct rl_state *state = bpf_map_lookup_elem(&rate_limit_state, &src_ip);
        if (!state) {
            struct rl_state new_state = {
                .window_start = now,
                .count = 1,
            };
            bpf_map_update_elem(&rate_limit_state, &src_ip, &new_state, BPF_ANY);
        } else {
            __u64 window_ns = (__u64)config->window_ms * 1000000;
            if (now - state->window_start >= window_ns) {
                state->window_start = now;
                state->count = 1;
            } else {
                state->count++;
                if (state->count > config->threshold) {
                    increment_counter(1);
                    emit_drop_event(&event, 2);
                    return XDP_DROP;
                }
            }
        }
    }

    increment_counter(0);
    return XDP_PASS;
}
