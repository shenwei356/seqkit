//go:build windows

package cmd

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
)

// fileIdentity uses the volume and file index rather than the mutable path.
func fileIdentity(path string, info os.FileInfo) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var id windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(windows.Handle(f.Fd()), &id); err != nil {
		return "", err
	}
	return fmt.Sprintf("%d:%d:%d", id.VolumeSerialNumber, id.FileIndexHigh, id.FileIndexLow), nil
}
