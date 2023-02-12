#ifndef __NEWPROCESS_
#define __NEWPROCESS_

#define MAX_FILENAME_LEN 127
#define MAX_ARGV_NUM 19
#define MAX_ARGV_LEN 127

struct process {
	int pid;
	int ppid;
	char filename[MAX_FILENAME_LEN];
	char argv[MAX_ARGV_NUM][MAX_ARGV_LEN];
};

struct exec_args {
	unsigned short type;
	unsigned char flags;
	unsigned char common_preempt_count;
	int common_pid;
	int __syscall_nr;
	const char * filename;
	const char *const * argv;
	const char *const *envp;
};

struct process_exit {
	int pid;
	int ppid;
	int error_code;
};

struct exit_args {
	unsigned short type;
	unsigned char flags;
	unsigned char common_preempt_count;
	int common_pid;
	int __syscall_nr;
	int error_code;
};


#endif
