// SPDX-License-Identifier: GPL-2.0
// Network monitoring eBPF program

#include <linux/bpf.h>
#include <linux/ptrace.h>
#include <linux/socket.h>
#include <linux/in.h>
#include <linux/in6.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>
#include <bpf/bpf_endian.h>

#define AF_INET    2
#define AF_INET6   10
#define TASK_COMM_LEN 16

struct conn_event {
    __u32 pid;
    __u32 uid;
    __u32 gid;
    char comm[TASK_COMM_LEN];
    __u16 family;
    __u16 sport;
    __u16 dport;
    __u32 saddr_v4;
    __u32 daddr_v4;
    __u8 saddr_v6[16];
    __u8 daddr_v6[16];
    __u64 timestamp;
    __u8 type; // 0=connect, 1=accept, 2=close
};

struct data_event {
    __u32 pid;
    __u64 bytes_sent;
    __u64 bytes_recv;
    __u64 timestamp;
};

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);
} conn_events SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);
} data_events SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10240);
    __type(key, __u32);
    __type(value, struct data_event);
} active_connections SEC(".maps");

static __always_inline void fill_conn_event(struct conn_event *event, __u16 family, __u8 type)
{
    __u64 pid_tgid = bpf_get_current_pid_tgid();
    __u64 uid_gid = bpf_get_current_uid_gid();

    event->pid = pid_tgid >> 32;
    event->uid = uid_gid;
    event->gid = uid_gid >> 32;
    event->family = family;
    event->type = type;
    event->timestamp = bpf_ktime_get_ns();
    bpf_get_current_comm(&event->comm, sizeof(event->comm));
}

SEC("kprobe/tcp_v4_connect")
int BPF_KPROBE(trace_tcp_v4_connect, struct sock *sk)
{
    struct conn_event *event;
    __u16 sport, dport;
    __u32 saddr, daddr;

    event = bpf_ringbuf_reserve(&conn_events, sizeof(*event), 0);
    if (!event)
        return 0;

    fill_conn_event(event, AF_INET, 0);

    BPF_CORE_READ_INTO(&sport, sk, __sk_common.skc_num);
    BPF_CORE_READ_INTO(&dport, sk, __sk_common.skc_dport);
    BPF_CORE_READ_INTO(&saddr, sk, __sk_common.skc_rcv_saddr);
    BPF_CORE_READ_INTO(&daddr, sk, __sk_common.skc_daddr);

    event->sport = sport;
    event->dport = bpf_ntohs(dport);
    event->saddr_v4 = saddr;
    event->daddr_v4 = daddr;

    bpf_ringbuf_submit(event, 0);
    return 0;
}

SEC("kprobe/tcp_v6_connect")
int BPF_KPROBE(trace_tcp_v6_connect, struct sock *sk)
{
    struct conn_event *event;
    __u16 sport, dport;

    event = bpf_ringbuf_reserve(&conn_events, sizeof(*event), 0);
    if (!event)
        return 0;

    fill_conn_event(event, AF_INET6, 0);

    BPF_CORE_READ_INTO(&sport, sk, __sk_common.skc_num);
    BPF_CORE_READ_INTO(&dport, sk, __sk_common.skc_dport);
    BPF_CORE_READ_INTO(&event->saddr_v6, sk, __sk_common.skc_v6_rcv_saddr.in6_u.u6_addr8);
    BPF_CORE_READ_INTO(&event->daddr_v6, sk, __sk_common.skc_v6_daddr.in6_u.u6_addr8);

    event->sport = sport;
    event->dport = bpf_ntohs(dport);

    bpf_ringbuf_submit(event, 0);
    return 0;
}

SEC("kprobe/inet_csk_accept")
int BPF_KPROBE(trace_inet_csk_accept, struct sock *sk)
{
    struct conn_event *event;
    __u16 family, sport, dport;

    BPF_CORE_READ_INTO(&family, sk, __sk_common.skc_family);

    if (family != AF_INET && family != AF_INET6)
        return 0;

    event = bpf_ringbuf_reserve(&conn_events, sizeof(*event), 0);
    if (!event)
        return 0;

    fill_conn_event(event, family, 1);

    BPF_CORE_READ_INTO(&sport, sk, __sk_common.skc_num);
    BPF_CORE_READ_INTO(&dport, sk, __sk_common.skc_dport);

    event->sport = sport;
    event->dport = bpf_ntohs(dport);

    if (family == AF_INET) {
        BPF_CORE_READ_INTO(&event->saddr_v4, sk, __sk_common.skc_rcv_saddr);
        BPF_CORE_READ_INTO(&event->daddr_v4, sk, __sk_common.skc_daddr);
    } else {
        BPF_CORE_READ_INTO(&event->saddr_v6, sk, __sk_common.skc_v6_rcv_saddr.in6_u.u6_addr8);
        BPF_CORE_READ_INTO(&event->daddr_v6, sk, __sk_common.skc_v6_daddr.in6_u.u6_addr8);
    }

    bpf_ringbuf_submit(event, 0);
    return 0;
}

SEC("kprobe/tcp_close")
int BPF_KPROBE(trace_tcp_close, struct sock *sk)
{
    struct conn_event *event;
    __u16 family;

    BPF_CORE_READ_INTO(&family, sk, __sk_common.skc_family);

    if (family != AF_INET && family != AF_INET6)
        return 0;

    event = bpf_ringbuf_reserve(&conn_events, sizeof(*event), 0);
    if (!event)
        return 0;

    fill_conn_event(event, family, 2);

    bpf_ringbuf_submit(event, 0);
    return 0;
}

char LICENSE[] SEC("license") = "GPL";
