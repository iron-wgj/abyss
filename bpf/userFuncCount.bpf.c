//+build ignore

#include "vmlinux.h"
#include "function.h"

// #include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

#ifdef asm_inline
#undef asm_inline
#define asm_inline asm
#endif

struct {
    __uint(type, BPF_MAP_TYPE_PERF_EVENT_ARRAY);
    __uint(key_size, sizeof(u32));
    __uint(value_size, sizeof(u32));
} events SEC(".maps");

long ringbuffer_flags = 0;

SEC("uprobe/func_call")
int uprobe__func_call(struct pt_regs *ctx)
{
	struct func_call fc = {};

	u64 id = bpf_get_current_pid_tgid();
	fc.pid = id >> 32;
	fc.tgid = (id << 32) >> 32;
	fc.ts = bpf_ktime_get_ns();

	bpf_perf_event_output(ctx, &events, BPF_F_CURRENT_CPU, &fc, sizeof(fc));
	return 0;
}

char _license[] SEC("license") = "GPL";
