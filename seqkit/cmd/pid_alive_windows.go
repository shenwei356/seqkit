//go:build windows

package cmd

import "golang.org/x/sys/windows"

// IsPidAlive checks process state and always releases the probe handle.
func IsPidAlive(pid int) bool {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err == windows.ERROR_ACCESS_DENIED {
		return true
	}
	if err == windows.ERROR_INVALID_PARAMETER || err == windows.ERROR_NOT_FOUND {
		return false
	}
	checkError(err)
	defer windows.CloseHandle(handle)
	var code uint32
	checkError(windows.GetExitCodeProcess(handle, &code))
	return code == 259 // STILL_ACTIVE, as defined by the Windows process API.
}
