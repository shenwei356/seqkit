//go:build !windows

package cmd

import "syscall"

// IsPidAlive treats a permission-denied probe as an existing process.
func IsPidAlive(pid int) bool {
	err := syscall.Kill(pid, syscall.Signal(0))
	switch err {
	case nil, syscall.EPERM:
		return true
	case syscall.ESRCH:
		return false
	default:
		checkError(err)
		return false
	}
}
