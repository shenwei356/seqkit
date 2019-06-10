package cmd

import (
	//	"gopkg.in/cheggaaa/pb.v2"
	"fmt"
	"os"
	"os/exec"
)

// Execute commands via bash.
func BashExec(command string) {
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed running command: %s - %s\n", command, err)
		os.Exit(1)
	}

}

// Get size of a file.
func FileSize(file string) int {
	info, err := os.Stat(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not stat file %s: %s\n", file, err)
		os.Exit(1)
	}
	return int(info.Size())
}
