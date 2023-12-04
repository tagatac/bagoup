// Copyright (C) 2022  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package scall

import "syscall"

//go:generate mockgen -destination=mock_scall/mock_scall.go github.com/tagatac/bagoup/v2/opsys/scall Syscall

type (
	// Syscall is a thin wrapper on the standard syscall library.
	Syscall interface {
		// Getrlimit wraps the system call by the same name.
		Getrlimit(which int, lim *syscall.Rlimit) error
		// Setrlimit wraps the system call by the same name.
		Setrlimit(which int, lim *syscall.Rlimit) error
	}
	sCall struct{}
)

// NewSyscall returns an instance of the Syscall wrapper.
func NewSyscall() Syscall                                    { return sCall{} }
func (sCall) Getrlimit(which int, lim *syscall.Rlimit) error { return syscall.Getrlimit(which, lim) }
func (sCall) Setrlimit(which int, lim *syscall.Rlimit) error { return syscall.Setrlimit(which, lim) }
