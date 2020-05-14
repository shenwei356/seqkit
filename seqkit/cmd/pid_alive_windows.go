// +build windows
package cmd

import "os"

func IsPidAlive(pid int) bool {
	_, err := os.FindProcess(pid)
	if err == nil {
		return true
	}
	return false
}
