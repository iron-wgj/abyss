//+build ignore

#include "vmlinux.h"
#include "function.h"
#include <bpf/bpf_helpers.h>

#ifdef asm_inline
#undef asm_inline
#define asm_inline asm
#endif

struct {
	__uint(type, BPF_MAP_TYPE_PERF_EVENT_ARRAY);
	__uint(key_size, sizeof(u32));
	__uint(value_size, sizeof(u32));
} func_ret SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 32);
	__type(key, struct func_dur_key);
	__type(value, u64);
} func_refs SEC(".maps");	


SEC("uprobe/func_entry")
int uprobe__func_entry(struct pt_regs *ctx) {
	struct func_dur_key fdk = {};
	u64 ts = bpf_ktime_get_ns();

	u64 id = bpf_get_current_pid_tgid();
	fdk.pid = id >> 32;
	fdk.tgid = (id << 32) >> 32;

	bpf_map_update_elem(&func_refs, &fdk, &ts,
			BPF_NOEXIST);
	return 0;
}

SEC("uprobe/func_exit")
int uprobe__func_exit(struct pt_regs *ctx) {
	struct func_dur_key fdk = {};
	u64 ts = bpf_ktime_get_ns();

	u64 id = bpf_get_current_pid_tgid();
	fdk.pid = id >> 32;
	fdk.tgid = (id << 32) >> 32;

	u64 *ets = bpf_map_lookup_elem(&func_refs, &fdk);
	if (ets == NULL) {
		return 0;
	}

	if (bpf_map_delete_elem(&func_refs, &fdk) < 0) {
		return 0;
	}
	struct func_dur fd = {
		.pid = fdk.pid,
		.tgid = fdk.tgid,
		.dur_ns = (ts - *ets),
	};

	bpf_perf_event_output(ctx, &func_ret,
			BPF_F_CURRENT_CPU, &fd, sizeof(fd));
	return 0;
}

char _license[] SEC("license") = "GPL";
