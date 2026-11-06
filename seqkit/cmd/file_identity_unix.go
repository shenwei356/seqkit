//go:build !windows

package cmd

import (
	"fmt"
	"os"
	"syscall"
)

// fileIdentity remains stable across renames, symlinks and hard links.
func fileIdentity(path string, info os.FileInfo) (string, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return resolvedPath(path)
	}
	return fmt.Sprintf("%d:%d", stat.Dev, stat.Ino), nil
}
