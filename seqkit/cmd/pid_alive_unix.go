package cmd

import "syscall"

func IsPidAlive(pid int) bool {
	killErr := syscall.Kill(pid, syscall.Signal(0))
	if killErr != nil {
		return true
	}

	return false
}
