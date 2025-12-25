package util

import (
	"fmt"
	"os"
)

var verbose bool

func SetVerbose(v bool) {
	verbose = v
}

func Logf(file *os.File, f string, args ...any) {
	if verbose {
		fmt.Fprintf(file, f, args...)
	}
}

func Logln(file *os.File, args ...any) {
	if verbose {
		fmt.Fprintln(file, args...)
	}
}
