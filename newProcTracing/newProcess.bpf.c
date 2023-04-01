//go:build ignore
#include"vmlinux.h"
#include<bpf/bpf_helpers.h>
#include<bpf/bpf_tracing.h>
#include<bpf/bpf_core_read.h>
#include"newProcess.h"

char LICENSE[] SEC("license") = "Dual BSD/GPL";

struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 256 * 1024);
} new_proc SEC(".maps");

const volatile unsigned long long max_argv_number = 19;

SEC("tp/syscalls/sys_enter_execve")
int handle_exec(struct exec_args *ctx)
{
	struct task_struct * task;
	struct process *proc;

	/* reverse sample from BPF ringbuf */
	proc = bpf_ringbuf_reserve(&new_proc, sizeof(*proc), 0);
	if (!proc)
		return 0;

	/* fill out the sample with data */
	proc->pid = bpf_get_current_pid_tgid() >> 32;

	task = (struct task_struct *)bpf_get_current_task();
	proc->ppid = BPF_CORE_READ(task, real_parent, tgid);
	
	bpf_probe_read_str(&proc->filename, sizeof(proc->filename),ctx->filename);
	
	for (int i = 1; i <= max_argv_number; i++)
	{
		const char *arg_ptr = NULL;
		long res = bpf_probe_read(&arg_ptr, sizeof(arg_ptr), &ctx->argv[i]);
		if (res != 0) break;
		bpf_probe_read_str(&proc->argv[i - 1], sizeof(proc->argv[i - 1]), arg_ptr);
	}

	/* successfully submit it to user-space for tracing argv of new peocess */
	bpf_ringbuf_submit(proc, 0);

	// bpf_printk("execve send a message, size %d.\n", sizeof(*proc));
	return 0;
}

/* maps and program used to monitor process exit */
struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 4*1024);
} exit_proc SEC(".maps");

SEC("tracepoint/sched/sched_process_exit")
int handle_exit(struct trace_event_raw_sched_process_template *args)
{
	struct task_struct * task;
	struct process_exit *e;
	
	/* reserve buffer in ringbuf */
	e = bpf_ringbuf_reserve(&exit_proc, sizeof(*e), 0);
	if (!e)
		return 0;
	
	/* fill out reserved ringbuf struct */
	e->pid = bpf_get_current_pid_tgid() >> 32;
	
	task = (struct task_struct *)bpf_get_current_task();
	e->ppid = BPF_CORE_READ(task, real_parent, tgid);

	e->error_code = BPF_CORE_READ(task, exit_code);
	e->error_code = e->error_code >> 8;

	/* submit ringbuf msg */
	bpf_ringbuf_submit(e, 0);
	
	// bpf_printk("Exit send a message.");
	return 0;
}
