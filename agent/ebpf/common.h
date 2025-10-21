// SPDX-License-Identifier: GPL-2.0
// Common definitions shared between eBPF and userspace

#ifndef __COMMON_H
#define __COMMON_H

#define TASK_COMM_LEN 16
#define MAX_FILENAME_LEN 128

// Connection event types
#define CONN_TYPE_CONNECT 0
#define CONN_TYPE_ACCEPT 1
#define CONN_TYPE_CLOSE 2

// Network connection event
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
    __u8 type;
};

// Data transfer event
struct data_event {
    __u32 pid;
    __u64 bytes_sent;
    __u64 bytes_recv;
    __u64 timestamp;
};

// Process exec event
struct exec_event {
    __u32 pid;
    __u32 ppid;
    __u32 uid;
    __u32 gid;
    char comm[TASK_COMM_LEN];
    char filename[MAX_FILENAME_LEN];
    char username[32];
    __u64 timestamp;
    __u32 cpu_id;
};

// Process exit event
struct exit_event {
    __u32 pid;
    __u32 ppid;
    __u32 uid;
    __u32 gid;
    char comm[TASK_COMM_LEN];
    __u64 timestamp;
    __u64 duration_ns;
    __u32 exit_code;
};

#endif /* __COMMON_H */
