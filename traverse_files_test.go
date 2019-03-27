package gotfp

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// Note that: fmt.Println() and fmt.Printf() are for debug.

func TestTraverseFiles(t *testing.T) {
	t.Log("Test start (log) (just show the output is not hang-up)")
	// fmt.Println("Test start (fmt) (just show the output is not hang-up)")
	root := "C:\\Go" // Set your root here.

	var counter uint64
	var depthLimitReachCounter uint64
	var depthLimit int = 100
	// Suppose the task here is find a file named "helloworld.go",
	//   which cannot be under the directory "./src" and its sub-directories.
	// The depth is limited to var depthLimit.
	//
	// You can set your task by editing following handler.
	handler := func(info *FInfo, depth int) Action {
		atomic.AddUint64(&counter, 1)
		if info == nil {
			return ActionContinue
		}
		if depth > depthLimit {
			t.Error("Fail to limit depth <=", depthLimit, ", depth =", depth)
			// fmt.Println("Fail to limit depth <=", depthLimit, ", depth =", depth)
		}
		if info.path == "" {
			t.Error("path is empty!")
			fmt.Println("path is empty!")
		}
		if info.err != nil {
			t.Log(info.err)
			// fmt.Println(info.err)
			return ActionContinue
		}
		if info.info == nil {
			t.Error("No error but info is nil!")
			// fmt.Println("No error but info is nil!")
			return ActionContinue
		}
		if depth >= depthLimit && info.info.IsDir() {
			atomic.AddUint64(&depthLimitReachCounter, 1)
			return ActionSkipDir
		}
		if info.info.Name() == "src" && info.info.IsDir() && depth == 1 {
			t.Logf("Skip %q (%q)", info.info.Name(), info.path)
			// fmt.Printf("Skip %q (%q)\n", info.info.Name(), info.path)
			return ActionSkipDir
		}
		if info.info.Name() == "helloworld.go" {
			t.Log("Found \"helloworld.go\". Size:", info.info.Size(), "bytes. Path:", info.path)
			// fmt.Println("Found \"helloworld.go\". It's size is", info.info.Size())
			return ActionExit
		}
		return ActionContinue
	}
	// End of handler definition.

	errChan := make(chan error, 10)
	doneChan := make(chan struct{})
	ticker := time.NewTicker(time.Second * 5)
	go func() {
		t.Log("Daemon start")
		// fmt.Println("Daemon start")
		defer t.Log("Daemon done")
		defer close(doneChan)
		for {
			select {
			case err, ok := <-errChan:
				if !ok {
					return
				}
				t.Error(err)
			case now := <-ticker.C:
				t.Log(now, "handler called", atomic.LoadUint64(&counter), "times")
				fmt.Println(now, "handler called", atomic.LoadUint64(&counter), "times")
			}
		}
	}()
	defer func() {
		ticker.Stop()
		t.Log("Wait for daemon stop")
		// fmt.Println("Wait for daemon stop")
		<-doneChan
		t.Log("Finally, handler called", atomic.LoadUint64(&counter),
			"times, depth limit (", depthLimit, ") reached",
			atomic.LoadUint64(&depthLimitReachCounter), "times.")
	}()

	err := TraverseFiles(root, handler, 8, errChan, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestTraverseFilesCompareWithWalk(t *testing.T) {
	// Use path/filepath.Walk() to do the same thing as above test.

	root := "C:\\Go" // Set your root here.

	var counter uint64

	walkFn := func(path string, info os.FileInfo, err error) error {
		atomic.AddUint64(&counter, 1)
		if err != nil {
			t.Log(err)
			return nil // Continue to search.
		}
		if info == nil {
			t.Error("No error but info is nil!")
			return nil // Continue to search.
		}
		if info.Name() == "src" && info.IsDir() && filepath.Join(root, "src") == path {
			t.Logf("Skip %q (%q)", info.Name(), path)
			return filepath.SkipDir
		}
		if info.Name() == "helloworld.go" {
			t.Log("Found \"helloworld.go\". Size:", info.Size(), "bytes. Path:", path)
			return filepath.SkipDir
		}
		return nil
	}

	daemonExitChan := make(chan struct{})
	doneChan := make(chan struct{})
	ticker := time.NewTicker(time.Second * 5)
	go func() {
		t.Log("Daemon start")
		// fmt.Println("Daemon start")
		defer t.Log("Daemon done")
		defer close(doneChan)
		for {
			select {
			case <-daemonExitChan:
				return
			case now := <-ticker.C:
				t.Log(now, "handler called", atomic.LoadUint64(&counter), "times")
				fmt.Println(now, "handler called", atomic.LoadUint64(&counter), "times")
			}
		}
	}()
	defer func() {
		ticker.Stop()
		t.Log("Wait for daemon stop")
		// fmt.Println("Wait for daemon stop")
		<-doneChan
		t.Log("Finally, handler called", atomic.LoadUint64(&counter), "times.")
	}()

	err := filepath.Walk(root, walkFn)
	close(daemonExitChan)
	if err != nil {
		t.Error(err)
	}
}

// TODO: Benchmark.
