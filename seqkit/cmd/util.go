// Copyright © 2019 Oxford Nanopore Technologies.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// BashExec executes a command via bash.
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

// FileSize gets size of a file by calling os.Stat.
func FileSize(file string) int {
	info, err := os.Stat(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not stat file %s: %s\n", file, err)
		os.Exit(1)
	}
	return int(info.Size())
}

// ReverseInt revsrees a slice of integers.
func ReverseInt(d []int) []int {
	s := make([]int, len(d))
	copy(d, s)
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

// FileExists checks if a file exists by calling os.Stat.
func FileExists(fn string) bool {
	_, err := os.Stat(fn)
	if err == nil {
		return true
	}
	return false
}

// MinInts calculates the minimum of a slice of integers.
func MinInts(s []int) (m int) {
	m = s[0]
	for _, e := range s {
		if e < m {
			m = e
		}
	}
	return
}

// MaxInts calculates the maximum of a slice of integers.
func MaxInts(s []int) (m int) {
	for _, e := range s {
		if e > m {
			m = e
		}
	}
	return
}

// SumInts calculates the sum of a slice of integers.
func SumInts(s []int) (r int) {
	for _, e := range s {
		r += e
	}
	return
}

// RevCompDNA reverse complements a DNA sequence string.
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
