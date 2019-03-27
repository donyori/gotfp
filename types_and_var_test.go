package gotfp

import (
	"errors"
	"fmt"
	"runtime"
)

// Interface to log in test.
// Use with testing.T or testing.B
type testLogger interface {
	Log(args ...interface{})
	Logf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Skip(args ...interface{})
	Skipf(format string, args ...interface{})
}

var (
	testRoot     string // Set as GOROOT.
	testMaxProcs int    // Set as GOMAXPROCS.
)

func init() {
	if testRoot == "" {
		testRoot = runtime.GOROOT()
		if testRoot == "" {
			_, filePath, _, ok := runtime.Caller(1)
			var err error
			if ok {
				err = fmt.Errorf("cannot get GOROOT, please set it manually in %q", filePath)
			} else {
				err = errors.New("cannot get GOROOT, please set it manually in test file.")
			}
			panic(err)
		}
	}
	if testMaxProcs == 0 {
		testMaxProcs = runtime.GOMAXPROCS(0) // Query GOMAXPROCS
		if testMaxProcs <= 0 {
			_, filePath, _, ok := runtime.Caller(1)
			var err error
			if ok {
				err = fmt.Errorf("cannot get GOMAXPROCS, please set it manually in %q", filePath)
			} else {
				err = errors.New("cannot get GOMAXPROCS, please set it manually in test file.")
			}
			panic(err)
		}
	}
}
