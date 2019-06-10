package cmd

import (
	//	"gopkg.in/cheggaaa/pb.v2"
	"fmt"
	"os"
	"os/exec"
	"strings"
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

// Reverse integer slice.
func ReverseInt(d []int) []int {
	s := make([]int, len(d))
	copy(d, s)
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

// Check if file exists.
func FileExists(fn string) bool {
	_, err := os.Stat(fn)
	if err == nil {
		return true
	}
	return false
}

// Get minimum of int slice.
func MinInts(s []int) (m int) {
	m = s[0]
	for _, e := range s {
		if e < m {
			m = e
		}
	}
	return
}

// Get maximum of int slice.
func MaxInts(s []int) (m int) {
	for _, e := range s {
		if e > m {
			m = e
		}
	}
	return
}

// Calculate sum of int slice.
func SumInts(s []int) (r int) {
	for _, e := range s {
		r += e
	}
	return
}

// Reverse complement DNA sequence.
func RevCompDNA(s string) string {
	size := len(s)
	s = strings.ToUpper(s)
	tmp := make([]byte, size)
	var inBase byte
	var outBase byte
	for i := 0; i < size; i++ {
		inBase = s[i]
		switch inBase {
		case 'A':
			outBase = 'T'
		case 'T':
			outBase = 'A'
		case 'G':
			outBase = 'C'
		case 'C':
			outBase = 'G'
		default:
			outBase = 'N'
		}
		tmp[size-1-i] = outBase
	}
	return string(tmp)
}
