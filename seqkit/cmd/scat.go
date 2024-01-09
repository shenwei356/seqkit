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
	"os"
	"os/signal"
	ospath "path"
	"regexp"
	"runtime"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/iafan/cwalk"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

type WatchCtrl int

type WatchCtrlChan chan WatchCtrl

// scatCmd represents the fish command
var scatCmd = &cobra.Command{
	GroupID: "basic",

	Use:   "scat",
	Short: "real time recursive concatenation and streaming of fastx files",
	Long:  "real time recursive concatenation and streaming of fastx files",

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
		gzOnly := getFlagBool(cmd, "gz-only")
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
		gzNr := 0
		if gzOnly {
			gzNr = 1
		}
		FASTA_REGEXP := fmt.Sprintf(".*\\.(fas|fa|fasta)(\\.gz){%d,1}$", gzNr)
		FASTQ_REGEXP := fmt.Sprintf(".*\\.(fastq|fq)(\\.gz){%d,1}$", gzNr)
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

// LaunchFxWatchers launches fastx watcher goroutines on multiple input directories.
func LaunchFxWatchers(dirs []string, ctrlChan WatchCtrlChan, re *regexp.Regexp, inFmt, outFmt string, qBase int, allowGaps bool, delta int, timeout string, dropString string, waitPid int, findOnly bool, outw *xopen.Writer) {
	allSeqChans := make([]chan *simpleSeq, len(dirs))
	allInCtrlChans := make([]WatchCtrlChan, len(dirs))
	allOutCtrlChans := make([]WatchCtrlChan, len(dirs))
	for i, dir := range dirs {
		allSeqChans[i] = make(chan *simpleSeq, 10000)
		allInCtrlChans[i] = make(WatchCtrlChan, 1000)
		allOutCtrlChans[i] = make(WatchCtrlChan, 0)
		go NewFxWatcher(dir, allSeqChans[i], allInCtrlChans[i], allOutCtrlChans[i], re, inFmt, outFmt, qBase, allowGaps, delta, dropString, findOnly)
	}
	sigChan := make(chan os.Signal, 1)
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
	QUITTING := false

MAIN:
	for {
		select {
		case <-sigChan:
			signal.Stop(sigChan)
			if !QUITTING {
				QUITTING = true
				sendQuitCmds()
				close(sigChan)
				for _ = range sigChan {
				}
			} else {
				signal.Reset(os.Interrupt)
				sigChan = nil
			}
			outw.Flush()
			continue MAIN
		case <-pidTimer.C:
			if !IsPidAlive(waitPid) {
				pidTimer.Stop()
				pidTimer.C = nil
				log.Info("Watched process with PID", waitPid, "exited.")
				sendQuitCmds()
			}
			outw.Flush()
			continue MAIN
		case <-ticker.C:
			ticker.Stop()
			ticker.C = nil
			log.Info("Inactivity limit of", timeout, "reached!")
			sendQuitCmds()
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
					case rawSeq := <-sc:
						if rawSeq == nil {
							log.Fatal("Trying to print nil sequence!")
						}
						switch rawSeq.Err {
						case nil:
							pass++
							outw.Write([]byte(rawSeq.Format(outFmt) + "\n"))
							outw.Flush()
						default:
							fail++
							os.Stderr.WriteString("From file: " + rawSeq.File + "\t" + rawSeq.String() + "\n")
						}
						if timeout != "" {
							ticker.Stop()
							ticker = time.NewTimer(td)
						}
						continue CHAN
					default:
						select {
						case fb := <-allOutCtrlChans[j]:
							if fb != WatchCtrl(-9) {
								log.Fatal("Invalid exit feedback:", fb)
							}

							allOutCtrlChans[j] = nil
							allInCtrlChans[j] = nil
							outw.Flush()
							activeCount--
							continue CHAN
						default:
							outw.Flush()
							continue CHAN
						}
					}
				} // select 1
			} // for chan
			if activeCount == 0 {
				outw.Flush()
				break MAIN
			}
		} // select 2
	} //for evers

	outw.Flush()
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

type WatchedFxPool struct {
	Map *sync.Map
}

func (m *WatchedFxPool) Range(f func(key, value interface{}) bool) {
	m.Map.Range(f)
}

func (m *WatchedFxPool) Get(k string) *WatchedFx {
	v, ok := m.Map.Load(k)
	if !ok {
		return nil
	}
	return v.(*WatchedFx)
}

func (m *WatchedFxPool) Insert(k string, v *WatchedFx) {
	m.Map.Store(k, v)
}

func (m *WatchedFxPool) Delete(k string) {
	m.Map.Delete(k)
}

func (m *WatchedFxPool) IsEmpty() bool {
	count := 0
	m.Map.Range(func(_, _ interface{}) bool {
		count++
		return false
	})
	return count == 0
}

type FxWatcher struct {
	Base string
	Pool *WatchedFxPool
}

// NewFxWatcher streams records from fastx files under a directory.
func NewFxWatcher(dir string, seqChan chan *simpleSeq, watcherCtrlChanIn, watcherCtrlChanOut WatchCtrlChan, re *regexp.Regexp, inFmt, outFmt string, qBase int, allowGaps bool, minDelta int, dropString string, findOnly bool) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("fsnotify error:", err)
	}
	defer watcher.Close()
	self := new(FxWatcher)
	self.Base = dir
	self.Pool = &WatchedFxPool{&sync.Map{}}
	dropDuration, err := time.ParseDuration(dropString)
	checkError(err)

	if findOnly {
		watcher.Close()
	}

	walkFunc := func(path string, info os.FileInfo, err error) error {
		path = ospath.Join(dir, path)
		wm := self.Pool.Get(path)
		if wm != nil {
			return nil
		}
		if info.IsDir() && !findOnly {
			err = watcher.Add(path)
			checkError(err)
			self.Pool.Insert(path, &WatchedFx{Name: path, IsDir: true})
			log.Info("Watching directory:", path)
			return nil
		} else {
			if !reFilterName(path, re) {
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
			self.Pool.Insert(path, &WatchedFx{Name: path, IsDir: false, SeqChan: sc, CtrlChanIn: ctrlIn, CtrlChanOut: ctrlOut, LastSize: fi.Size(), LastTry: created})

			ctrlIn <- StreamTry
			if findOnly {
				log.Info("Streaming file:", path)
				ctrlIn <- StreamQuit
			} else {
				log.Info("Watching file:", path)
			}
			return nil
		}
		return nil
	}

	go func() {
	SFOR:
		for {
			select {
			case <-watcherCtrlChanIn:
				log.Info("Exiting...")
				sigChan := make(chan os.Signal, 2)
				signal.Notify(sigChan, os.Interrupt)
				self.Pool.Range(func(k, v interface{}) bool {
					ePath := k.(string)
					w := v.(*WatchedFx)
					watcher.Remove(ePath)
					if w.IsDir {
						self.Pool.Delete(ePath)
						log.Info("Stopped watching directory: ", ePath)
						return true
					}
					log.Info("Stopped watching file: ", ePath)
					if !findOnly {
						w.CtrlChanIn <- StreamQuit
					}
				DRAIN:
					for fb := range w.CtrlChanOut {
						switch fb {
						case StreamExited:
							self.Pool.Delete(ePath)
							break DRAIN
						case StreamEOF:
							continue DRAIN
						default:
							log.Fatal("Invalid feedback when trying to quit:", int(fb))
						}
					}
					return true
				})
				watcherCtrlChanOut <- WatchCtrl(-9)
				return
			case event := <-watcher.Events:
				ePath := event.Name
				if event.Op&fsnotify.Remove == fsnotify.Remove {
					di := self.Pool.Get(ePath)
					if di == nil {
						continue SFOR
					}
					if di.IsDir {
						log.Info("Removed directory:", ePath)
						watcher.Remove(ePath)
						self.Pool.Delete(ePath)
						continue SFOR
					}
					log.Info("Removed file:", ePath)
					di.CtrlChanIn <- StreamQuit
					fb := <-di.CtrlChanOut
					if fb != StreamExited {
						log.Fatal("Invalid removal feedback:", int(fb))
					}
					self.Pool.Delete(ePath)
					continue SFOR
				}
				if event.Op&fsnotify.Rename == fsnotify.Rename {
					wm := self.Pool.Get(ePath)
					if wm != nil {
						log.Info("Stopped watching renamed file:", ePath)
						wm.CtrlChanIn <- StreamQuit
						fb := <-wm.CtrlChanOut
						if fb != StreamExited {
							log.Fatal("Invalid renaming feedback:", int(fb))
						}
						self.Pool.Delete(ePath)
					}
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					wm := self.Pool.Get(ePath)
					if wm != nil {
						continue
					}
					fi, err := os.Stat(ePath)
					checkError(err)
					if fi.IsDir() {
						log.Info("Watching new directory:", ePath)
						err := watcher.Add(ePath)
						checkError(err)
						self.Pool.Insert(ePath, &WatchedFx{Name: ePath, IsDir: true})
						err = cwalk.Walk(dir, walkFunc)
						checkError(err)
						continue SFOR

					}
					if !reFilterName(ePath, re) {
						continue SFOR
					}
					created := time.Now()

					sc := seqChan
					ctrlIn, ctrlOut := NewRawSeqStreamFromFile(ePath, sc, qBase, inFmt, allowGaps)
					self.Pool.Insert(ePath, &WatchedFx{Name: ePath, IsDir: false, SeqChan: sc, CtrlChanIn: ctrlIn, CtrlChanOut: ctrlOut, LastSize: fi.Size(), LastTry: created})
					err = watcher.Add(ePath)
					checkError(err)
					log.Info("Watching new file:", ePath)
					ctrlIn <- StreamTry
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					fi, err := os.Stat(ePath)
					checkError(err)
					wm := self.Pool.Get(ePath)
					if wm == nil || fi.IsDir() {
						continue SFOR
					}
					if time.Now().Sub(wm.LastTry) < dropDuration {
						continue SFOR
					}
					if wm.LastSize == fi.Size() {
						continue SFOR
					}
					delta := fi.Size() - wm.LastSize
					if delta < 0 {
						log.Info("Stopped watching truncated file:", ePath)
						wm := self.Pool.Get(ePath)
						wm.CtrlChanIn <- StreamQuit
						fb := <-wm.CtrlChanOut
						if fb != StreamExited {
							log.Fatal("Invalid truncation feedback:", int(fb))
						}
						self.Pool.Delete(ePath)
						continue SFOR
					}
					if delta < int64(minDelta) {
						continue SFOR
					}

					wm.CtrlChanIn <- StreamTry
					wm.LastTry = time.Now()

				}
			case err, ok := <-watcher.Errors:
				if ok {
					log.Fatalf("fsnotify error:", err)
				}
			default:
				time.Sleep(NAP_SLEEP)
			}
		}
	}()

	if !findOnly {
		err = watcher.Add(dir)
		checkError(err)
	}
	self.Pool.Insert(dir, &WatchedFx{Name: dir, IsDir: true})
	if !findOnly {
		log.Info(fmt.Sprintf("Watcher (%s) launched on root: %s", inFmt, dir))
	} else {
		log.Info(fmt.Sprintf("Streaming %s records from directory: %s", inFmt, dir))
	}

	err = cwalk.Walk(dir, walkFunc)
	checkError(err)

	if findOnly {
		watcherCtrlChanIn <- WatchCtrl(1)
	}

	for {
		if self.Pool.IsEmpty() {
			break
		}
		for {
			time.Sleep(NAP_SLEEP)

			self.Pool.Range(func(k, v interface{}) bool {
				ePath := k.(string)
				w := v.(*WatchedFx)
				select {
				case seq := <-w.SeqChan:
					seqChan <- seq
					return true
				case fb := <-w.CtrlChanOut:
					switch fb {
					case StreamExited:
						self.Pool.Delete(ePath)
						return true
					case StreamEOF:
						return true
					default:
						log.Fatal("Invalid feedback on channel: ", int(fb))
					}
				}
				return true
			})

		}
	}

}

func init() {
	RootCmd.AddCommand(scatCmd)

	scatCmd.Flags().StringP("regexp", "r", "", "regexp for watched files, by default guessed from the input format")
	scatCmd.Flags().StringP("format", "i", "fastq", "input and output format: fastq or fasta (fastq)")
	scatCmd.Flags().StringP("in-format", "I", "", "input format: fastq or fasta (fastq)")
	scatCmd.Flags().StringP("out-format", "O", "", "output format: fastq or fasta")
	scatCmd.Flags().BoolP("allow-gaps", "A", false, "allow gap character (-) in sequences")
	scatCmd.Flags().BoolP("find-only", "f", false, "concatenate existing files and quit")
	scatCmd.Flags().BoolP("gz-only", "g", false, "only look for gzipped files (.gz suffix)")
	scatCmd.Flags().StringP("time-limit", "T", "", "quit after inactive for this time period")
	scatCmd.Flags().IntP("wait-pid", "p", -1, "after process with this PID exited")
	scatCmd.Flags().IntP("delta", "d", 5, "minimum size increase in kilobytes to trigger parsing")
	scatCmd.Flags().StringP("drop-time", "D", "500ms", "Notification drop interval")
	scatCmd.Flags().IntP("qual-ascii-base", "b", 33, "ASCII BASE, 33 for Phred+33")
}
