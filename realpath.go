package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var paramQuiet bool = false
var paramLoops int = 1000

// isTTY attempts to determine whether the current stdout refers to a terminal.
func isTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting Stat of os.Stdout:", err)
		return true // Assume human readable
	}
	return (fi.Mode() & os.ModeNamedPipe) != os.ModeNamedPipe
}

func init() {
	flag.BoolVar(&paramQuiet, "q", paramQuiet, "Whether to stifle warnings when realpath fails.")
	flag.IntVar(&paramLoops, "l", paramLoops, "The maximum number of symlink eval loops to try.")
}

// canonicalize attempts to return the canonical path for a given path.
//
// It will follow symlinks if loops is > 0, and only attempt to follow a
// symlink that many times. If a cycle is detected, the loop is terminated
// early and the most recently evaluated path is returned.
//
// If the input path does not exist, it will be converted to an absolute path
// and returned.
//
// If an error is encountered at any point, it will return.
func canonicalize(p string, loops int) (string, error) {
	var orig string = p
	var err error
	var lp string
	var tested = make(map[string]bool)
	var loop int = 0

	for lp != p {
		lp = p
		p, err = filepath.Abs(p)
		if err != nil {
			return lp, err
		}

		loop++
		if loop > loops {
			if loops > 0 {
				err = fmt.Errorf("looped too many times canonicalizing %q", orig)
			}
			return p, err
		}

		lp = p

		var fi os.FileInfo
		fi, err = os.Lstat(p)
		if err != nil && !os.IsNotExist(err) {
			return p, err
		}

		if !os.IsNotExist(err) && fi.Mode()&os.ModeSymlink == os.ModeSymlink {
			if tested[p] {
				err = fmt.Errorf("symlink loop encountered with path %q on loop %d", orig, loop)
				break
			}

			p, err = os.Readlink(p)
			if os.IsNotExist(err) {
				p = lp
			}
		} else {
			p = lp
		}

		if os.IsNotExist(err) {
			err = nil
		}

		tested[lp] = true
	}

	return p, err
}

func main() {
	flag.Parse()

	args := flag.Args()
	// default to working directory if no paths given
	if len(args) == 0 {
		wd, err := os.Getwd()
		if err != nil {
			if !paramQuiet {
				panic(err)
			}
			os.Exit(1)
		}
		args = []string{wd}
	}

	// accumulate paths and convert them to absolute paths
	paths := make([]string, len(args))
	for i, p := range args {
		canon, err := canonicalize(p, paramLoops)
		paths[i] = canon
		if err != nil && !paramQuiet {
			log.Println("Error:", err)
		}
	}

	io.WriteString(os.Stdout, strings.Join(paths, "\n"))
	if isTTY() {
		io.WriteString(os.Stdout, "\n")
	}
}
