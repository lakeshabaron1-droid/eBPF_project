#ifndef __BPF_ENDIAN_H
#define __BPF_ENDIAN_H

#include <linux/types.h>

#if __BYTE_ORDER__ == __ORDER_LITTLE_ENDIAN__

#define __bpf_ntohs(x) __builtin_bswap16(x)
#define __bpf_htons(x) __builtin_bswap16(x)
#define __bpf_ntohl(x) __builtin_bswap32(x)
#define __bpf_htonl(x) __builtin_bswap32(x)

#elif __BYTE_ORDER__ == __ORDER_BIG_ENDIAN__

#define __bpf_ntohs(x) (x)
#define __bpf_htons(x) (x)
#define __bpf_ntohl(x) (x)
#define __bpf_htonl(x) (x)

#else
#error "Unknown byte order"
#endif

#define bpf_htons(x) ((__be16)__bpf_htons(x))
#define bpf_ntohs(x) ((__u16)__bpf_ntohs(x))
#define bpf_htonl(x) ((__be32)__bpf_htonl(x))
#define bpf_ntohl(x) ((__u32)__bpf_ntohl(x))

#endif
