package gotfp

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// Task:
// Find a file named "helloworld.go" under GOROOT and its sub-directories,
//   which cannot be under the directory "$GOROOT/src" and its sub-directories.

func TestFindFile(t *testing.T) {
	var counter uint64
	handler := testFindFileMakeHandler(t, &counter)
	errChan := make(chan error, 10)
	doneChan := make(chan struct{})
	ticker := time.NewTicker(time.Second)
	go func() {
		// t.Log("Daemon start")
		fmt.Println("Daemon start")
		// defer t.Log("Daemon done")
		defer fmt.Println("Daemon done")
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
		// t.Log("Wait for daemon stop")
		fmt.Println("Wait for daemon stop")
		<-doneChan
		t.Log("Finally, handler called", atomic.LoadUint64(&counter), "times")
	}()

	defer close(errChan)
	err := TraverseFiles(testRoot, handler, testMaxProcs, errChan, time.Microsecond)
	if err != nil {
		t.Error(err)
	}
}

// Use path/filepath.Walk() to do the same thing as above test.
func TestFindFileWithWalk(t *testing.T) {
	var counter uint64
	walkFn := testFindFileMakeWalkFn(t, testRoot, &counter)
	daemonExitChan := make(chan struct{})
	doneChan := make(chan struct{})
	ticker := time.NewTicker(time.Second)
	go func() {
		// t.Log("Daemon start")
		fmt.Println("Daemon start")
		// defer t.Log("Daemon done")
		defer fmt.Println("Daemon done")
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
		// t.Log("Wait for daemon stop")
		fmt.Println("Wait for daemon stop")
		<-doneChan
		t.Log("Finally, handler called", atomic.LoadUint64(&counter), "times.")
	}()

	err := filepath.Walk(testRoot, walkFn)
	close(daemonExitChan)
	if err != nil {
		t.Error(err)
	}
}

func BenchmarkFindFile(b *testing.B) {
	fmt.Println("GOMAXPROCS =", testMaxProcs)
	errChan := make(chan error, 10)
	doneChan := make(chan struct{})
	benchmarks := []struct {
		name         string
		workerNumber int
	}{
		{"wn=1", 1},
		{"wn=2", 2},
		{"wn=3", 3},
		{"wn=4", 4},
		{"wn=GOMAXPROCS/2", testMaxProcs / 2},
		{"wn=GOMAXPROCS", testMaxProcs},
		{"wn=GOMAXPROCS*2", testMaxProcs * 2},
		{"wn=GOMAXPROCS*4", testMaxProcs * 4},
		// ...
	}
	go func() {
		// b.Log("Daemon start")
		fmt.Println("Daemon start")
		// defer b.Log("Daemon done")
		defer fmt.Println("Daemon done")
		defer close(doneChan)
		for err := range errChan {
			b.Error(err)
		}
	}()
	defer func() {
		// b.Log("Wait for daemon stop")
		fmt.Println("Wait for daemon stop")
		<-doneChan
	}()
	defer close(errChan)
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			handler := testFindFileMakeHandler(b, nil)
			var err error
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err = TraverseFiles(testRoot, handler,
					bm.workerNumber, errChan, 0)
				if err != nil {
					b.Error(err)
				}
			}
		})
	}
	// Compare with path/filepath.Walk()
	b.Run("Walk()", func(b *testing.B) {
		walkFn := testFindFileMakeWalkFn(b, testRoot, nil)
		var err error
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err = filepath.Walk(testRoot, walkFn)
			if err != nil {
				b.Error(err)
			}
		}
	})
}

func testFindFileMakeHandler(tb testing.TB, counter *uint64) FileHandler {
	return func(info *FInfo, depth int) Action {
		if counter != nil {
			atomic.AddUint64(counter, 1)
		}
		if info == nil {
			tb.Error("info is nil!")
			return ActionContinue
		}
		if info.path == "" {
			tb.Error("path is empty!")
		}
		if info.err != nil {
			// tb.Log(info.err)
			return ActionContinue
		}
		if info.info == nil {
			tb.Error("No error but info is nil!")
			return ActionContinue
		}
		if info.info.Name() == "src" && info.info.IsDir() && depth == 1 {
			// tb.Log("Skip", info.path)
			return ActionSkipDir
		}
		if info.info.Name() == "helloworld.go" {
			// tb.Log("Found \"helloworld.go\". Size:", info.info.Size(), "bytes. Path:", info.path)
			return ActionExit
		}
		return ActionContinue
	}
}

func testFindFileMakeWalkFn(tb testing.TB, root string, counter *uint64) filepath.WalkFunc {
	isFound := false
	return func(path string, info os.FileInfo, err error) error {
		if counter != nil {
			atomic.AddUint64(counter, 1)
		}
		if isFound {
			return filepath.SkipDir
		}
		if err != nil {
			// tb.Log(err)
			return nil // Continue to search.
		}
		if info == nil {
			tb.Error("No error but info is nil!")
			return nil // Continue to search.
		}
		if info.Name() == "src" && info.IsDir() && filepath.Join(root, "src") == path {
			// tb.Log("Skip", path)
			return filepath.SkipDir
		}
		if info.Name() == "helloworld.go" {
			isFound = true
			// tb.Log("Found \"helloworld.go\". Size:", info.Size(), "bytes. Path:", path)
			return filepath.SkipDir
		}
		return nil
	}
}
