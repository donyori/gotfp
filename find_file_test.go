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
	handler := testFindFileMakeFileHandler(t, &counter)
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
	TraverseFiles(handler, testMaxProcs,
		errChan, time.Microsecond, testRoot)
}

func TestFindFileWithBatch(t *testing.T) {
	var counter uint64
	handler := testFindFileMakeBatchHandler(t, &counter)
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
	TraverseBatches(handler, testMaxProcs,
		errChan, time.Microsecond, testRoot)
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
		nameSuffix   string
		workerNumber int
	}{
		{"wn=1", 1},
		{"wn=2", 2},
		{"wn=3", 3},
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
		b.Run("f-"+bm.nameSuffix, func(b *testing.B) {
			handler := testFindFileMakeFileHandler(b, nil)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				TraverseFiles(handler, bm.workerNumber,
					errChan, 0, testRoot)
			}
		})
	}
	for _, bm := range benchmarks {
		b.Run("b-"+bm.nameSuffix, func(b *testing.B) {
			handler := testFindFileMakeBatchHandler(b, nil)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				TraverseBatches(handler, bm.workerNumber,
					errChan, 0, testRoot)
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

func testFindFileMakeFileHandler(tb testing.TB, counter *uint64) FileHandler {
	isFound := false
	return func(info FInfo, depth int) Action {
		if counter != nil {
			atomic.AddUint64(counter, 1)
		}
		if isFound {
			return ActionExit
		}
		if info.Path == "" {
			tb.Error("path is empty!")
		}
		if info.Err != nil {
			// tb.Log(info.Err)
			return ActionContinue
		}
		if info.Info == nil {
			tb.Error("No error but info is nil!")
			return ActionContinue
		}
		if info.Info.Name() == "src" && info.Info.IsDir() && depth == 1 {
			// tb.Log("Skip", info.Path)
			return ActionSkipDir
		}
		if info.Info.Name() == "helloworld.go" {
			isFound = true
			// tb.Log("Found \"helloworld.go\". Size:", info.Info.Size(), "bytes. Path:", info.Path)
			return ActionExit
		}
		return ActionContinue
	}
}

func testFindFileMakeBatchHandler(tb testing.TB, counter *uint64) BatchHandler {
	isFound := false
	return func(batch Batch, depth int) (
		action Action, skipDirs map[string]bool) {
		if counter != nil {
			atomic.AddUint64(counter, 1)
		}
		if isFound {
			return ActionExit, nil
		}
		if batch.Parent.Path == "" {
			tb.Error("Parent path is empty!")
		}
		if batch.Parent.Err != nil {
			return ActionContinue, nil
		}
		if batch.Parent.Info == nil {
			tb.Error("No error but info is nil!")
		}
		for i := range batch.RegFiles {
			if batch.RegFiles[i].Info.Name() == "helloworld.go" {
				isFound = true
				return ActionExit, nil
			}
		}
		if depth == 0 {
			for i := range batch.Dirs {
				if batch.Dirs[i].Info.Name() == "src" {
					return ActionSkipDir, map[string]bool{
						batch.Dirs[i].Path: true,
					}
				}
			}
		}
		return ActionContinue, nil
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
