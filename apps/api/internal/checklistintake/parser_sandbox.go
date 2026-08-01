package checklistintake

import (
	"errors"
	"strings"
)

type ParserSandboxPolicy struct {
	SeccompProfilePath string
	AllowedSyscalls    []string
	Environment        []string
	NetworkDisabled    bool
	FilesystemReadOnly bool
}

func DefaultParserSandboxPolicy() ParserSandboxPolicy {
	return ParserSandboxPolicy{
		SeccompProfilePath: "deploy/local/checklist-pdf-parser-seccomp.json",
		AllowedSyscalls:    []string{"read", "write", "close", "fstat", "mmap", "munmap", "brk", "rt_sigaction", "rt_sigprocmask", "exit", "exit_group", "openat", "newfstatat", "lseek", "readlink", "getrandom", "clock_gettime"},
		NetworkDisabled:    true,
		FilesystemReadOnly: true,
	}
}

func (policy ParserSandboxPolicy) Validate() error {
	if !policy.NetworkDisabled || !policy.FilesystemReadOnly {
		return errors.New("parser sandbox must disable network and writes")
	}
	for _, syscall := range policy.AllowedSyscalls {
		switch strings.TrimSpace(syscall) {
		case "socket", "connect", "accept", "accept4", "bind", "listen", "sendto", "recvfrom", "execve", "execveat":
			return errors.New("parser sandbox permits a prohibited capability")
		}
	}
	for _, variable := range policy.Environment {
		upper := strings.ToUpper(variable)
		for _, secret := range []string{"SECRET", "PASSWORD", "TOKEN", "DATABASE_URL", "API_KEY"} {
			if strings.Contains(upper, secret) {
				return errors.New("parser sandbox environment contains secret material")
			}
		}
	}
	return nil
}
