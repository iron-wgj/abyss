#include "vmlinux.h"

struct func_call {
	u32 pid;
	u32 tgid;
	u64 ts;
};

struct func_dur_key {
	u32 pid;
	u32 tgid;
};

struct func_dur {
	u32 pid;
	u32 tgid;
	u64 dur_ns;
};
