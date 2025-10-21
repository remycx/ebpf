// SPDX-License-Identifier: GPL-2.0
// Process monitoring eBPF program

#include <linux/bpf.h>
#include <linux/ptrace.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

#define TASK_COMM_LEN 16
#define MAX_FILENAME_LEN 128
#define MAX_ARGS 5
#define ARG_LEN 64

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

struct process_info {
    __u64 start_time;
    __u32 ppid;
    __u32 uid;
    __u32 gid;
    char comm[TASK_COMM_LEN];
};

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);
} exec_events SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);
} exit_events SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10240);
    __type(key, __u32);
    __type(value, struct process_info);
} process_start SEC(".maps");

SEC("tp/sched/sched_process_exec")
int handle_exec(struct trace_event_raw_sched_process_exec *ctx)
{
    struct task_struct *task;
    struct exec_event *event;
    struct process_info info = {};
    __u64 pid_tgid, uid_gid;
    __u32 pid;

    event = bpf_ringbuf_reserve(&exec_events, sizeof(*event), 0);
    if (!event)
        return 0;

    pid_tgid = bpf_get_current_pid_tgid();
    uid_gid = bpf_get_current_uid_gid();
    pid = pid_tgid >> 32;

    event->pid = pid;
    event->uid = uid_gid;
    event->gid = uid_gid >> 32;
    event->timestamp = bpf_ktime_get_ns();
    event->cpu_id = bpf_get_smp_processor_id();

    bpf_get_current_comm(&event->comm, sizeof(event->comm));

    task = (struct task_struct *)bpf_get_current_task();
    BPF_CORE_READ_INTO(&event->ppid, task, real_parent, tgid);

    // Read filename from tracepoint args
    bpf_probe_read_kernel_str(&event->filename, sizeof(event->filename),
                             (void *)ctx->filename);

    // Store process start info for exit tracking
    info.start_time = event->timestamp;
    info.ppid = event->ppid;
    info.uid = event->uid;
    info.gid = event->gid;
    bpf_probe_read_kernel(&info.comm, sizeof(info.comm), &event->comm);
    bpf_map_update_elem(&process_start, &pid, &info, BPF_ANY);

    bpf_ringbuf_submit(event, 0);
    return 0;
}

SEC("tp/sched/sched_process_exit")
int handle_exit(struct trace_event_raw_sched_process_template *ctx)
{
    struct task_struct *task;
    struct exit_event *event;
    struct process_info *info;
    __u64 pid_tgid, uid_gid, current_time;
    __u32 pid;

    pid_tgid = bpf_get_current_pid_tgid();
    pid = pid_tgid >> 32;

    // Only track processes we've seen exec
    info = bpf_map_lookup_elem(&process_start, &pid);
    if (!info)
        return 0;

    event = bpf_ringbuf_reserve(&exit_events, sizeof(*event), 0);
    if (!event)
        goto cleanup;

    uid_gid = bpf_get_current_uid_gid();
    current_time = bpf_ktime_get_ns();

    event->pid = pid;
    event->uid = uid_gid;
    event->gid = uid_gid >> 32;
    event->timestamp = current_time;
    event->duration_ns = current_time - info->start_time;
    event->ppid = info->ppid;

    bpf_probe_read_kernel(&event->comm, sizeof(event->comm), &info->comm);

    task = (struct task_struct *)bpf_get_current_task();
    BPF_CORE_READ_INTO(&event->exit_code, task, exit_code);

    bpf_ringbuf_submit(event, 0);

cleanup:
    bpf_map_delete_elem(&process_start, &pid);
    return 0;
}

SEC("tp/sched/sched_process_fork")
int handle_fork(struct trace_event_raw_sched_process_fork *ctx)
{
    // Track forks for better parent-child relationships
    // Future enhancement: emit fork events
    return 0;
}

char LICENSE[] SEC("license") = "GPL";
