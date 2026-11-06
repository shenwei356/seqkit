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
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/shenwei356/go-logging"
	"github.com/shenwei356/xopen"
	"github.com/spf13/cobra"
)

type WatchCtrl int

type WatchCtrlChan chan WatchCtrl

// watchSequenceQueueSize bounds pending records across all watched directories.
const watchSequenceQueueSize = 128

// newWatchSequenceChannels gives every directory access to one shared queue.
func newWatchSequenceChannels(n int) []chan *simpleSeq {
	queue := make(chan *simpleSeq, watchSequenceQueueSize)
	channels := make([]chan *simpleSeq, n)
	for i := range channels {
		channels[i] = queue
	}
	return channels
}

// scatCmd represents the fish command
var scatCmd = &cobra.Command{
	GroupID: "basic",

	Use:   "scat",
	Short: "real time recursive concatenation and streaming of fastx files",
	Long: `real time recursive concatenation and streaming of fastx files

Attention:
  1. --infile-list accepts directories, one per line, and is merged with positional
     directories. Overlapping directories and file aliases are processed once.
  2. Renames within the watched directories retain the read position.
  3. Complete appended records are flushed promptly; incomplete records wait
     for more data.
  4. In --find-only mode, file errors do not stop other files, but the final exit
     status is nonzero.

`,

	Run: func(cmd *cobra.Command, args []string) {
		config := getConfigs(cmd)
		outFile := config.OutFile
		runtime.GOMAXPROCS(config.Threads)

		if config.Quiet {
			logging.SetLevel(logging.ERROR, "seqkit")
		}
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
		if inFmt == "" || outFmt == "" {
			log.Fatal("Invalid format specified: empty effective format")
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
		compressionSuffix := `(\.(gz|xz|zst|bz2))?`
		if gzOnly {
			compressionSuffix = `\.gz`
		}
		FASTA_REGEXP := `(?i)\.(fasta|faa|fas|fna|fa)` + compressionSuffix + `$`
		FASTQ_REGEXP := `(?i)\.(fastq|fq)` + compressionSuffix + `$`
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

		dirs := getFileListFromArgsAndFile(cmd, args, true, "infile-list", !config.SkipFileCheck)
		roots := make([]string, 0, len(dirs))
		seenDirs := make(map[string]bool)
		for _, dir := range dirs {
			if isStdin(dir) {
				continue
			}
			real, err := filepath.EvalSymlinks(dir)
			checkError(err)
			real, err = filepath.Abs(real)
			checkError(err)
			info, err := os.Stat(real)
			checkError(err)
			if !info.IsDir() {
				checkError(fmt.Errorf("not a directory: %s", dir))
			}
			if !seenDirs[real] {
				roots = append(roots, filepath.Clean(dir))
				seenDirs[real] = true
			}
		}
		dirs = roots
		outfh, err := xopen.Wopen(outFile)
		checkError(err)
		defer closeOutput(outfh)
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
		err = LaunchFxWatchers(dirs, ctrlChan, reFilter, inFmt, outFmt, qBase, allowGaps, delta, timeLimit, dropString, waitPid, findOnly, outfh)
		closeOutput(outfh)
		checkError(err)

	},
}

// trackedFx retains parser state and physical identity across path changes.
// lastSize is the size at the last parse attempt; pendingSize is the latest
// notification size. busy lasts until EOF feedback; quitting drains exactly once.
type trackedFx struct {
	key, name                     string
	info                          fs.FileInfo
	in, feedback                  chan SeqStreamCtrl
	lastSize, pendingSize         int64
	lastTry                       time.Time
	busy, pending, quitting, done bool
}

// LaunchFxWatchers coordinates all roots, so aliases never launch duplicate parsers.
func LaunchFxWatchers(dirs []string, ctrlChan WatchCtrlChan, re *regexp.Regexp, inFmt, outFmt string, qBase int, allowGaps bool, delta int, timeout, dropString string, waitPid int, findOnly bool, outw *xopen.Writer) error {
	drop, err := time.ParseDuration(dropString)
	if err != nil {
		return err
	}
	var idle time.Duration
	if timeout != "" {
		idle, err = time.ParseDuration(timeout)
		if err != nil {
			return err
		}
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()
	sequences := newWatchSequenceChannels(1)[0]
	files := make(map[string]*trackedFx)
	running := make(map[string]*trackedFx)
	var queued []*trackedFx
	active, nextQueued := 0, 0
	// Scan-only readers finish, so a bounded worker set avoids exhausting descriptors.
	scanLimit := max(1, min(16, runtime.GOMAXPROCS(0)))
	watchedDirs := make(map[string]bool)
	hadError := false
	fileError := func(path string, err error) { log.Errorf("%s: %v", path, err); hadError = true }
	start := func(item *trackedFx) {
		item.in, item.feedback = NewRawSeqStreamFromFile(item.name, sequences, qBase, inFmt, allowGaps)
		item.busy, item.quitting, item.lastTry = true, findOnly, time.Now()
		running[item.key] = item
		if findOnly {
			log.Info("Streaming file:", item.name)
			item.in <- StreamQuit
		} else {
			log.Info("Watching file:", item.name)
			item.in <- StreamTry
		}
	}

	add := func(path string, info fs.FileInfo) {
		key, err := fileIdentity(path, info)
		if err != nil {
			fileError(path, err)
			return
		}
		if previous := files[key]; previous != nil {
			if current, err := os.Stat(previous.name); err != nil || !os.SameFile(current, previous.info) {
				previous.name = path
			}
			return
		}
		item := &trackedFx{key: key, name: path, info: info, lastSize: info.Size()}
		files[key] = item
		active++
		if findOnly {
			queued = append(queued, item)
		} else {
			start(item)
		}

	}
	scan := func() {
		for _, root := range dirs {
			walkRoot, err := filepath.EvalSymlinks(root)
			if err != nil {
				fileError(root, err)
				continue
			}
			walkErr := filepath.WalkDir(walkRoot, func(path string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					fileError(path, walkErr)
					return nil
				}
				if entry.IsDir() {
					if !findOnly && !watchedDirs[path] {
						if err := watcher.Add(path); err != nil {
							fileError(path, err)
						} else {
							watchedDirs[path] = true
						}
					}
					return nil
				}
				if !reFilterName(path, re) {
					return nil
				}
				info, err := os.Stat(path)
				if err != nil {
					fileError(path, err)
					return nil
				}
				if !info.IsDir() {
					add(path, info)
				}
				return nil
			})
			if walkErr != nil {
				fileError(root, walkErr)
			}
		}
	}
	scan()
	quit := func(item *trackedFx) {
		if !item.done && !item.quitting {
			item.quitting = true
			if item.in == nil {
				item.done = true
				active--
			} else {
				item.in <- StreamQuit
			}
		}
	}
	quitting := findOnly
	stopping := false
	quitAll := func() {
		quitting, stopping = true, true
		for _, item := range files {
			quit(item)
		}
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	defer signal.Stop(signals)
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	pidTick := time.NewTicker(500 * time.Millisecond)
	defer pidTick.Stop()
	var pidEvents <-chan time.Time
	if waitPid >= 0 {
		pidEvents = pidTick.C
	}
	lastActivity := time.Now()
	pass, discarded := 0, 0
	for {
		if findOnly && !stopping {
			for len(running) < scanLimit && nextQueued < len(queued) {
				item := queued[nextQueued]
				queued[nextQueued] = nil
				nextQueued++
				if !item.done {
					start(item)
				}
			}
		}

		if quitting && active == 0 && len(sequences) == 0 {
			break
		}
		select {
		case record := <-sequences:
			lastActivity = time.Now()
			if record.Err == nil {
				pass++
				if _, err := outw.WriteString(record.Format(outFmt) + "\n"); err != nil {
					return err
				}
				if err := outw.Flush(); err != nil {
					return err
				}
			} else if fileErr, fatal := record.Err.(*rawFileError); fatal {
				fileError(fileErr.file, fileErr.cause)
			} else {
				discarded++
				fmt.Fprintf(os.Stderr, "From file: %s\t%s\n", record.File, record.String())
			}
		case <-signals:
			quitAll()
		case <-pidEvents:
			if !IsPidAlive(waitPid) {
				pidEvents = nil
				quitAll()
			}
		case event := <-watcher.Events:
			if quitting {
				continue
			}
			if event.Op&(fsnotify.Create|fsnotify.Rename|fsnotify.Remove) != 0 {
				// Relocate a renamed identity before deciding that it left the watched roots.
				scan()
				for _, item := range files {
					if item.done || item.quitting {
						continue
					}
					current, err := os.Stat(item.name)
					if err != nil || !os.SameFile(current, item.info) {
						quit(item)
					}
				}
			}
			if event.Op&fsnotify.Write != 0 {
				info, err := os.Stat(event.Name)
				if os.IsNotExist(err) {
					continue
				}
				if err != nil {
					fileError(event.Name, err)
					continue
				}
				if info.IsDir() {
					continue
				}
				key, err := fileIdentity(event.Name, info)
				if err != nil {
					fileError(event.Name, err)
					continue
				}
				if item := files[key]; item != nil && !item.done && !item.quitting {
					item.pending, item.pendingSize = true, info.Size()
				}
			}
		case err := <-watcher.Errors:
			if err != nil {
				fileError("fsnotify", err)
				quitAll()
			}
		case <-tick.C:
			for _, item := range running {
				if item.done {
					continue
				}
				select {
				case feedback := <-item.feedback:
					if feedback == StreamExited {
						item.done = true
						active--
						delete(running, item.key)
						item.in, item.feedback = nil, nil
						if findOnly {
							delete(files, item.key)
						}
					} else if feedback == StreamEOF {
						item.busy = false
					}
				default:
				}
				if !item.done && !item.quitting && !item.busy && item.pending && time.Since(item.lastTry) >= drop {
					if item.pendingSize < item.lastSize {
						quit(item)
						continue
					}
					if item.pendingSize-item.lastSize >= int64(delta) {
						item.in <- StreamTry
						item.lastTry, item.lastSize, item.busy = time.Now(), item.pendingSize, true
					}
					item.pending = false
				}
			}
			if !quitting && idle > 0 && time.Since(lastActivity) >= idle {
				quitAll()
			}
		}
	}
	log.Infof("Total stats:\tPass records: %d\tDiscarded lines: %d", pass, discarded)
	if hadError {
		return fmt.Errorf("one or more input files could not be read")
	}
	return outw.Flush()
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
	scatCmd.Flags().IntP("delta", "d", 5, "minimum size increase in kilobytes (base 1024) to trigger parsing")
	scatCmd.Flags().StringP("drop-time", "D", "500ms", "Notification drop interval")
	scatCmd.Flags().IntP("qual-ascii-base", "b", 33, "ASCII BASE, 33 for Phred+33")
}
