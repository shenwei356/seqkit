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
	"github.com/fsnotify/fsnotify"
	"github.com/iafan/cwalk"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	//"path"
	//"path/filepath"
	"runtime"
	"sync"
	"time"
)

type WatchCtrl int

type WatchCtrlChan chan WatchCtrl

// scatCmd represents the fish command
var scatCmd = &cobra.Command{
	Use:   "scat",
	Short: "look for short sequences in larger sequences using local alignment",
	Long:  "look for short sequences in larger sequences using local alignment",

	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		outFile := config.OutFile
		runtime.GOMAXPROCS(config.Threads)

		qBase := getFlagPositiveInt(cmd, "qual-ascii-base")
		inFmt := getFlagString(cmd, "in-format")
		outFmt := getFlagString(cmd, "out-format")
		dropString := getFlagString(cmd, "drop-time")
		allowGaps := getFlagBool(cmd, "allow-gaps")
		timeLimit := getFlagFloat64(cmd, "time-limit")
		waitPid := getFlagInt(cmd, "wait-pid")
		delta := getFlagInt(cmd, "delta") * 1024
		regexp := getFlagString(cmd, "regexp")

		dirs := getFileList(args, true)
		var err error
		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Close()
		ctrlChan := make(WatchCtrlChan)
		ndirs := []string{}
		for _, d := range dirs {
			if d != "-" {
				ndirs = append(ndirs, d)
			}
		}
		if len(ndirs) == 0 {
			log.Info("No directories given to watch! Exiting.")
			os.Exit(1)
		}
		LaunchFxWatchers(dirs, ctrlChan, regexp, inFmt, outFmt, qBase, allowGaps, delta, dropString, timeLimit, waitPid)

	},
}

func LaunchFxWatchers(dirs []string, ctrlChan WatchCtrlChan, regexp string, inFmt, outFmt string, qBase int, allowGaps bool, delta int, dropString string, timeLimit float64, waitPid int) {
	for _, dir := range dirs {
		go NewFxWatcher(dir, ctrlChan, regexp, inFmt, outFmt, qBase, allowGaps, delta, dropString)
	}
	sigChan := make(chan os.Signal, 5)
	signal.Notify(sigChan, os.Interrupt)

	for {
		select {
		case <-sigChan:
			signal.Stop(sigChan)
			for i, _ := range dirs {
				ctrlChan <- WatchCtrl(i)
				<-ctrlChan
			}
			return
		default:
			time.Sleep(BIG_SLEEP)
		}
	}
}

type WatchedFx struct {
	Name      string
	LastSize  int64
	LastTry   time.Time
	BytesRead int64
	IsDir     bool
	SeqChan   chan *simpleSeq
	CtrlChan  chan SeqStreamCtrl
}

type WatchedFxPool map[string]*WatchedFx

type FxWatcher struct {
	Base  string
	Pool  WatchedFxPool
	Mutex sync.Mutex
}

func NewFxWatcher(dir string, ctrlChan WatchCtrlChan, regexp string, inFmt, outFmt string, qBase int, allowGaps bool, minDelta int, dropString string) {
	sigChan := make(chan os.Signal, 5)
	signal.Notify(sigChan, os.Interrupt)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("fsnotify error:", err)
	}
	defer watcher.Close()
	self := new(FxWatcher)
	self.Base = dir
	self.Pool = make(WatchedFxPool)
	dropDuration, err := time.ParseDuration(dropString)
	checkError(err)

	go func() {
	SFOR:
		for {
			select {
			case <-ctrlChan:
				self.Mutex.Lock()
				if len(self.Pool) > 0 {
					for ePath, w := range self.Pool {
						watcher.Remove(ePath)
						if w.IsDir {
							delete(self.Pool, ePath)
							continue
						}
						log.Info("Stopped watching: ", ePath)
						w.CtrlChan <- StreamQuit
						<-w.CtrlChan
						delete(self.Pool, ePath)
					}
				}
				self.Mutex.Unlock()
				log.Info("Exiting.")
				ctrlChan <- WatchCtrl(-9)
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				ePath := event.Name
				if event.Op&fsnotify.Remove == fsnotify.Remove {
					self.Mutex.Lock()
					di := self.Pool[ePath]
					if di == nil {
						self.Mutex.Unlock()
						continue SFOR
					}
					if di.IsDir {
						log.Info("Removed directory:", event.Name)
						watcher.Remove(event.Name)
						delete(self.Pool, ePath)
						self.Mutex.Unlock()
						continue SFOR
					}
					log.Info("Removed file:", event.Name)
					di.CtrlChan <- StreamQuit
					<-di.CtrlChan
					delete(self.Pool, ePath)
					self.Mutex.Unlock()
					continue SFOR
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					fi, err := os.Stat(ePath)
					checkError(err)
					self.Mutex.Lock()
					if self.Pool[ePath] == nil {
						self.Mutex.Unlock()
						continue SFOR
					}
					if time.Now().Sub(self.Pool[ePath].LastTry) < dropDuration {
						self.Mutex.Unlock()
						continue SFOR
					}
					if self.Pool[ePath].LastSize == fi.Size() {
						self.Mutex.Unlock()
						continue SFOR
					}
					delta := fi.Size() - self.Pool[ePath].LastSize
					if delta < 0 {
						log.Info("Stopped watching truncated file:", ePath)
						self.Pool[ePath].CtrlChan <- StreamQuit
						<-self.Pool[ePath].CtrlChan
						delete(self.Pool, ePath)
						self.Mutex.Unlock()
						continue SFOR
					}
					if delta < int64(minDelta) {
						self.Mutex.Unlock()
						continue SFOR
					}
					time.Sleep(NAP_SLEEP)
					self.Pool[ePath].CtrlChan <- StreamTry
					self.Pool[ePath].LastTry = time.Now()
					self.Mutex.Unlock()

				} else if event.Op&fsnotify.Create == fsnotify.Create {
					fi, err := os.Stat(ePath)
					checkError(err)
					self.Mutex.Lock()
					if fi.IsDir() {
						log.Info("New directory:", event.Name)
						watcher.Add(ePath)
						self.Pool[ePath] = &WatchedFx{Name: ePath, IsDir: true}
						self.Mutex.Unlock()
						continue SFOR
					}
					created := time.Now()
					log.Info("new file:", ePath)

					sc, ctrl := NewRawSeqStreamFromFile(ePath, 1000, qBase, inFmt, allowGaps)

					time.Sleep(NAP_SLEEP)
					ctrl <- StreamTry
					self.Pool[ePath] = &WatchedFx{Name: ePath, IsDir: false, SeqChan: sc, CtrlChan: ctrl, LastSize: fi.Size(), LastTry: created}
					watcher.Add(ePath)
					self.Mutex.Unlock()
				}
				if event.Op&fsnotify.Rename == fsnotify.Rename {
					log.Info("Stopped watching renamed file:", ePath)
					self.Pool[ePath].CtrlChan <- StreamQuit
					<-self.Pool[ePath].CtrlChan
					delete(self.Pool, ePath)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Fatalf("fsnotify error:", err)
			default:

			}
		}
	}()

	self.Mutex.Lock()
	err = watcher.Add(dir)
	self.Pool[dir] = &WatchedFx{Name: dir, IsDir: true}
	checkError(err)
	log.Info(fmt.Sprintf("Watcher (%s) launched on root: %s", inFmt, dir))

	walkFunc := func(path string, info os.FileInfo, err error) error {
		log.Info("New entry:", path)
		return nil
	}
	err = cwalk.Walk(dir, walkFunc)
	checkError(err)

	self.Mutex.Unlock()

	for {
		for ePath, w := range self.Pool {
			self.Mutex.Lock()
		CHAN:
			for {
				select {
				case seq, ok := <-w.SeqChan:
					if !ok {
						break CHAN
					}
					fmt.Println(seq)
				case e, ok := <-w.CtrlChan:
					if !ok {
						break CHAN
					}
					if e == StreamEOF {
						break CHAN
					}
					if e == StreamExited {
						delete(self.Pool, ePath)
						break CHAN
					}
					if e == StreamTry {
						w.CtrlChan <- e
						time.Sleep(BIG_SLEEP)
						break CHAN
					}
				default:
					break CHAN
				}
			}
			self.Mutex.Unlock()
			fi, err := os.Stat(ePath)
			if err != nil {
				w.LastSize = fi.Size()
			}
		}

		if len(self.Pool) == 0 {
			ctrlChan <- -1
			return
		}
	}

}

func init() {
	RootCmd.AddCommand(scatCmd)

	scatCmd.Flags().StringP("regexp", "r", ".*\\.(fastq|fq)", "output format: auto, fq, fas")
	scatCmd.Flags().StringP("in-format", "I", "fastq", "input format: fastq or fasta (fastq)")
	scatCmd.Flags().StringP("out-format", "O", "fastq", "output format: fastq or fasta (fastq)")
	scatCmd.Flags().BoolP("allow-gaps", "A", false, "allow gap character (-) in sequences")
	scatCmd.Flags().Float64P("time-limit", "T", -1, "quit after inactive for this many hours")
	scatCmd.Flags().IntP("wait-pid", "p", -1, "after process with this PID exited")
	scatCmd.Flags().IntP("delta", "d", 2, "minimum size increase to trigger parsing (5 kilobytes)")
	scatCmd.Flags().StringP("drop-time", "D", "500ms", "Notification drop interval (500ms)")
	scatCmd.Flags().IntP("qual-ascii-base", "b", 33, "ASCII BASE, 33 for Phred+33")
}
