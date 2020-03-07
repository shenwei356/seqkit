// Copyright © 2020 Botond Sipos.
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
	ospath "path"
	"regexp"
	"runtime"
	"sync"
	"syscall"
	"time"
)

type WatchCtrl int

type WatchCtrlChan chan WatchCtrl

func reFilterName(name string, re *regexp.Regexp) bool {
	return re.MatchString(name)
}

func checkFileFormat(format string) {
	switch format {
	case "fasta":
	case "fastq":
	case "":
	default:
		log.Fatal("Invalid format specified:", format)
	}
}

// scatCmd represents the fish command
var scatCmd = &cobra.Command{
	Use:   "scat",
	Short: "look for short sequences in larger sequences using local alignment",
	Long:  "look for short sequences in larger sequences using local alignment",

	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		outFile := config.OutFile
		runtime.GOMAXPROCS(config.Threads)

		quiet := config.Quiet // FIXME: add quiet mode
		_ = quiet
		qBase := getFlagPositiveInt(cmd, "qual-ascii-base")
		inFmt := getFlagString(cmd, "in-format")
		checkFileFormat(inFmt)
		outFmt := getFlagString(cmd, "out-format")
		checkFileFormat(outFmt)
		inOutFmt := getFlagString(cmd, "format")
		checkFileFormat(inOutFmt)
		if inFmt == "" {
			inFmt = inOutFmt
		}
		if outFmt == "" {
			outFmt = inOutFmt
		}
		log.Info("Input format is:", inFmt)
		log.Info("Output format is:", outFmt)

		dropString := getFlagString(cmd, "drop-time")
		allowGaps := getFlagBool(cmd, "allow-gaps")
		timeLimit := getFlagString(cmd, "time-limit")
		waitPid := getFlagInt(cmd, "wait-pid")
		findOnly := getFlagBool(cmd, "find-only")
		delta := getFlagInt(cmd, "delta") * 1024
		reStr := getFlagString(cmd, "regexp")
		var err error
		FASTA_REGEXP := ".*\\.(fas|fa|fasta)$"
		FASTQ_REGEXP := ".*\\.(fastq|fq)$"
		if reStr == "" {
			switch inFmt {
			case "fasta":
				reStr = FASTA_REGEXP
			case "fastq":
				reStr = FASTQ_REGEXP
			default:
				log.Fatal("Impossible input format:", inFmt)
			}
		}
		if findOnly {
			log.Info("Will stream files matching regexp: \"" + reStr + "\"")
		} else {
			log.Info("Will watch files matching regexp: \"" + reStr + "\"")
		}
		reFilter, err := regexp.Compile(reStr)
		checkError(err)

		dirs := getFileList(args, true)
		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer outfh.Flush()
		defer outfh.Close()
		ctrlChan := make(WatchCtrlChan, 0)
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
		LaunchFxWatchers(dirs, ctrlChan, reFilter, inFmt, outFmt, qBase, allowGaps, delta, timeLimit, dropString, waitPid, findOnly, outfh)

	},
}

func LaunchFxWatchers(dirs []string, ctrlChan WatchCtrlChan, re *regexp.Regexp, inFmt, outFmt string, qBase int, allowGaps bool, delta int, timeout string, dropString string, waitPid int, findOnly bool, outw *xopen.Writer) {
	allSeqChans := make([]chan *simpleSeq, len(dirs))
	allInCtrlChans := make([]WatchCtrlChan, len(dirs))
	allOutCtrlChans := make([]WatchCtrlChan, len(dirs))
	for i, dir := range dirs {
		allSeqChans[i] = make(chan *simpleSeq, 0)
		allInCtrlChans[i] = make(WatchCtrlChan, 0)
		allOutCtrlChans[i] = make(WatchCtrlChan, 0)
		go NewFxWatcher(dir, allSeqChans[i], allInCtrlChans[i], allOutCtrlChans[i], re, inFmt, outFmt, qBase, allowGaps, delta, dropString, findOnly)
	}
	sigChan := make(chan os.Signal, 5)
	signal.Notify(sigChan, os.Interrupt)
	defer outw.Flush()

	pidTimer := *time.NewTicker(time.Millisecond * 500)
	if waitPid < 0 {
		pidTimer.C = nil
	} else {
		log.Info("Running until process with PID", waitPid, "exits.")
	}
	ticker := time.NewTimer(time.Minute)
	ticker.Stop()
	td := time.Duration(0)
	if timeout != "" {
		var err error
		td, err = time.ParseDuration(timeout)
		checkError(err)
		ticker = time.NewTimer(td)
		log.Info("Will exit after being inactive for", timeout)
	}

	pass, fail := 0, 0

	sendQuitCmds := func() {
		for i, cc := range allInCtrlChans {
			if cc == nil {
				continue
			}
			cc <- WatchCtrl(i)
		}
	}

	activeCount := 0

	QUITING := false

MAIN:
	for {
		select {
		case <-sigChan:
			if !QUITING {
				QUITING = true
				signal.Stop(sigChan)
				sigChan = nil
				sendQuitCmds()
			}
			continue MAIN
		case <-pidTimer.C:
			killErr := syscall.Kill(waitPid, syscall.Signal(0))
			if killErr != nil && !QUITING {
				QUITING = true
				pidTimer.Stop()
				pidTimer.C = nil
				log.Info("Watched process with PID", waitPid, "exited.")
				sendQuitCmds()
			}
			continue MAIN
		case <-ticker.C:
			if !QUITING {
				QUITING = true
				ticker.Stop()
				ticker.C = nil
				log.Info("Inactivity limit of", timeout, "reached!")
				sendQuitCmds()
			}
			continue MAIN
		default:
			activeCount = 0
		CHAN:
			for j, sc := range allSeqChans {
				if allInCtrlChans[j] == nil || allOutCtrlChans[j] == nil {
					continue CHAN
				}
				activeCount++
				for {
					select {
					case rawSeq, ok := <-sc:
						if !ok {
							continue
						}
						if rawSeq == nil {
							log.Fatal("Trying to print nil sequence!")
						}
						switch rawSeq.Err {
						case nil:
							pass++
							outw.Write([]byte(rawSeq.Format(outFmt) + "\n"))
						default:
							fail++
							os.Stderr.WriteString("From file: " + rawSeq.File + "\t" + rawSeq.String() + "\n")
						}
						if timeout != "" {
							ticker.Stop()
							ticker = time.NewTimer(td)
						}
						continue
					case fb, ok := <-allOutCtrlChans[j]:
						if !ok || allOutCtrlChans[j] == nil {
							continue CHAN
						}
						if fb != WatchCtrl(-9) {
							log.Fatal("Invalid exit feedback:", fb)
						}

						allOutCtrlChans[j] = nil
						allInCtrlChans[j] = nil
						outw.Flush()
						activeCount--
						continue CHAN
					default:
						continue CHAN
					}
				} // select 1
			} // for chan
			if activeCount == 0 {
				outw.Flush()
				break MAIN
			}
		} // select 2
	} //for evers

	log.Info(fmt.Sprintf("Total stats:\tPass records: %d\tDiscarded lines: %d\n", pass, fail))
}

type WatchedFx struct {
	Name        string
	LastSize    int64
	LastTry     time.Time
	BytesRead   int64
	IsDir       bool
	SeqChan     chan *simpleSeq
	CtrlChanIn  chan SeqStreamCtrl
	CtrlChanOut chan SeqStreamCtrl
}

type WatchedFxPool map[string]*WatchedFx

type FxWatcher struct {
	Base  string
	Pool  WatchedFxPool
	Mutex sync.Mutex
}

func NewFxWatcher(dir string, seqChan chan *simpleSeq, watcherCtrlChanIn, watcherCtrlChanOut WatchCtrlChan, re *regexp.Regexp, inFmt, outFmt string, qBase int, allowGaps bool, minDelta int, dropString string, findOnly bool) {
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

	if findOnly {
		watcher.Close()
	}

	walkFunc := func(path string, info os.FileInfo, err error) error {
		path = ospath.Join(dir, path)
		self.Mutex.Lock()
		if self.Pool[path] != nil {
			self.Mutex.Unlock()
			return nil
		}
		if info.IsDir() && !findOnly {
			err = watcher.Add(path)
			checkError(err)
			self.Pool[path] = &WatchedFx{Name: path, IsDir: true}
			log.Info("Watching directory:", path)
			self.Mutex.Unlock()
			return nil
		} else {
			if !reFilterName(path, re) {
				self.Mutex.Unlock()
				return nil
			}
			created := time.Now()
			sc := seqChan
			ctrlIn, ctrlOut := NewRawSeqStreamFromFile(path, sc, qBase, inFmt, allowGaps)
			if !findOnly {
				err := watcher.Add(path)
				checkError(err)
			}
			fi, err := os.Stat(path)
			checkError(err)
			self.Pool[path] = &WatchedFx{Name: path, IsDir: false, SeqChan: sc, CtrlChanIn: ctrlIn, CtrlChanOut: ctrlOut, LastSize: fi.Size(), LastTry: created}

			if findOnly {
				log.Info("Streaming file:", path)
				ctrlIn <- StreamQuit
			} else {
				log.Info("Watching file:", path)
				ctrlIn <- StreamTry
			}
			self.Mutex.Unlock()
			return nil
		}
		return nil
	}

	go func() {
	SFOR:
		for {
			select {
			case _, ok := <-watcherCtrlChanIn:
				if !ok {
					continue
				}
				log.Info("Exiting...")
				sigChan := make(chan os.Signal, 2)
				signal.Notify(sigChan, os.Interrupt)
				self.Mutex.Lock()
				for ePath, w := range self.Pool {
					watcher.Remove(ePath)
					if w.IsDir {
						delete(self.Pool, ePath)
						log.Info("Stopped watching directory: ", ePath)
						continue
					}
					log.Info("Stopped watching file: ", ePath)
					if !findOnly {
						w.CtrlChanIn <- StreamQuit
					}
				DRAIN:
					for fb := range w.CtrlChanOut {
						switch fb {
						case StreamExited:
							delete(self.Pool, ePath)
							break DRAIN
						case StreamEOF:
							continue DRAIN
						default:
							log.Fatal("Invalid feedback when trying to quit:", int(fb))
						}
					}
				}
				self.Mutex.Unlock()
				watcherCtrlChanOut <- WatchCtrl(-9)
				return
			case event, ok := <-watcher.Events:
				if !ok {
					continue
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
						log.Info("Removed directory:", ePath)
						watcher.Remove(ePath)
						delete(self.Pool, ePath)
						self.Mutex.Unlock()
						continue SFOR
					}
					log.Info("Removed file:", ePath)
					di.CtrlChanIn <- StreamQuit
					fb := <-di.CtrlChanOut
					if fb != StreamExited {
						log.Fatal("Invalid removal feedback:", int(fb))
					}
					delete(self.Pool, ePath)
					self.Mutex.Unlock()
					continue SFOR
				}
				if event.Op&fsnotify.Rename == fsnotify.Rename {
					log.Info("Stopped watching renamed file:", ePath)
					self.Pool[ePath].CtrlChanIn <- StreamQuit
					fb := <-self.Pool[ePath].CtrlChanOut
					if fb != StreamExited {
						log.Fatal("Invalid renaming feedback:", int(fb))
					}
					self.Mutex.Lock()
					delete(self.Pool, ePath)
					self.Mutex.Unlock()
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					if self.Pool[ePath] != nil {
						continue
					}
					fi, err := os.Stat(ePath)
					checkError(err)
					self.Mutex.Lock()
					if fi.IsDir() {
						log.Info("Watching new directory:", ePath)
						err := watcher.Add(ePath)
						checkError(err)
						self.Pool[ePath] = &WatchedFx{Name: ePath, IsDir: true}
						self.Mutex.Unlock()
						err = cwalk.Walk(dir, walkFunc)
						checkError(err)
						continue SFOR

					}
					if !reFilterName(ePath, re) {
						self.Mutex.Unlock()
						continue SFOR
					}
					created := time.Now()

					sc := seqChan
					ctrlIn, ctrlOut := NewRawSeqStreamFromFile(ePath, sc, qBase, inFmt, allowGaps)
					self.Pool[ePath] = &WatchedFx{Name: ePath, IsDir: false, SeqChan: sc, CtrlChanIn: ctrlIn, CtrlChanOut: ctrlOut, LastSize: fi.Size(), LastTry: created}
					err = watcher.Add(ePath)
					checkError(err)
					log.Info("Watching new file:", ePath)
					ctrlIn <- StreamTry
					self.Mutex.Unlock()
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					fi, err := os.Stat(ePath)
					checkError(err)
					self.Mutex.Lock()
					if self.Pool[ePath] == nil || fi.IsDir() {
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
						self.Pool[ePath].CtrlChanIn <- StreamQuit
						fb := <-self.Pool[ePath].CtrlChanOut
						if fb != StreamExited {
							log.Fatal("Invalid truncation feedback:", int(fb))
						}
						delete(self.Pool, ePath)
						self.Mutex.Unlock()
						continue SFOR
					}
					if delta < int64(minDelta) {
						self.Mutex.Unlock()
						continue SFOR
					}
					self.Pool[ePath].CtrlChanIn <- StreamTry
					self.Pool[ePath].LastTry = time.Now()
					self.Mutex.Unlock()

				}
			case err, ok := <-watcher.Errors:
				if ok {
					log.Fatalf("fsnotify error:", err)
				}
			default:
				time.Sleep(time.Microsecond)
			}
		}
	}()

	self.Mutex.Lock()
	if !findOnly {
		err = watcher.Add(dir)
		checkError(err)
	}
	self.Pool[dir] = &WatchedFx{Name: dir, IsDir: true}
	if !findOnly {
		log.Info(fmt.Sprintf("Watcher (%s) launched on root: %s", inFmt, dir))
	} else {
		log.Info(fmt.Sprintf("Streaming %s records from directory: %s", inFmt, dir))
	}
	self.Mutex.Unlock()

	err = cwalk.Walk(dir, walkFunc)
	checkError(err)

	if findOnly {
		watcherCtrlChanIn <- WatchCtrl(1)
	}

	for {
		self.Mutex.Lock()
		if len(self.Pool) == 0 {
			break
		}
	POOL:
		for ePath, w := range self.Pool {
			for {
				select {
				case seq, ok := <-w.SeqChan:
					if !ok {
						continue POOL
					}
					seqChan <- seq
				case fb, ok := <-w.CtrlChanOut:
					if !ok {
						break POOL
					}
					switch fb {
					case StreamExited:
						delete(self.Pool, ePath)
						continue POOL
					case StreamEOF:
						continue POOL
					default:
						log.Fatal("Invalid feedback on channel: ", int(fb))
					}

				default:
					continue POOL
				}
			}
		}
		self.Mutex.Unlock()
	}

}

func init() {
	RootCmd.AddCommand(scatCmd)

	scatCmd.Flags().StringP("regexp", "r", "", "regexp for watched files, by default guessed from the input format")
	scatCmd.Flags().StringP("format", "i", "fastq", "input and output format: fastq or fasta (fastq)")
	scatCmd.Flags().StringP("in-format", "I", "", "input format: fastq or fasta (fastq)")
	scatCmd.Flags().StringP("out-format", "O", "", "output format: fastq or fasta")
	scatCmd.Flags().BoolP("allow-gaps", "A", false, "allow gap character (-) in sequences")
	scatCmd.Flags().BoolP("find-only", "f", false, "concatenate exisiting files and quit")
	scatCmd.Flags().StringP("time-limit", "T", "", "quit after inactive for this time period")
	scatCmd.Flags().IntP("wait-pid", "p", -1, "after process with this PID exited")
	scatCmd.Flags().IntP("delta", "d", 5, "minimum size increase in kilobytes to trigger parsing")
	scatCmd.Flags().StringP("drop-time", "D", "500ms", "Notification drop interval")
	scatCmd.Flags().IntP("qual-ascii-base", "b", 33, "ASCII BASE, 33 for Phred+33")
}
