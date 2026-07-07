#ifndef __BPF_HELPERS_H
#define __BPF_HELPERS_H

#ifndef SEC
#define SEC(name) \

    _Pragma("GCC diagnostic push")                      \
    _Pragma("GCC diagnostic ignored \"-Wignored-attributes\"") \
    __attribute__((section(name), used))                 \
    _Pragma("GCC diagnostic pop")

#endif


#define __uint(name, val)  int (*name)[val]
#define __type(name, val)  typeof(val) *name
#define __array(name, val) typeof(val) *name[]



static void *(*bpf_map_lookup_elem)(void *map, const void *key) = (void *) 1;


static long (*bpf_map_update_elem)(void *map, const void *key,
                                   const void *value, __u64 flags) = (void *) 2;


static long (*bpf_map_delete_elem)(void *map, const void *key) = (void *) 3;

static __u64 (*bpf_ktime_get_ns)(void) = (void *) 5;


static __u32 (*bpf_get_prandom_u32)(void) = (void *) 7;



static void *(*bpf_ringbuf_reserve)(void *ringbuf, __u64 size,
                                     __u64 flags) = (void *) 131;
static void (*bpf_ringbuf_submit)(void *data, __u64 flags) = (void *) 132;



static void (*bpf_ringbuf_discard)(void *data, __u64 flags) = (void *) 133;



#ifndef BPF_ANY
#define BPF_ANY       0  /* Create or update */
