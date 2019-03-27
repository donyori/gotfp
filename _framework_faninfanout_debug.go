package gotfp

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/donyori/gocontainer"
	"github.com/donyori/gocontainer/pqueue"
	"github.com/donyori/gorecover"
)

func mainProc(root string, handler workerHandler,
	workerNumber int, workerErrChan chan<- error,
	workerSendErrTimeout time.Duration) error {
	if handler == nil {
		return ErrNilHandler
	}
	if workerNumber <= 0 {
		return ErrNonPositiveWorkerNumber
	}
	pq, err := pqueue.NewPriorityQueue(workerNumber*8, false)
	if err != nil {
		return err
	}
	// Set first sub-task, the root.
	topSubTask := &subTask{
		fileInfo: FInfo{path: root},
		depth:    0,
	}
	err = pq.Enqueue(topSubTask)
	if err != nil {
		return err
	}

	// For debug.
	var activeCounter int32
	var handlingCounter int32

	// runningWg for check whether workers are exited or not.
	// subTaskWg for check sub-tasks are done or not.
	var runningWg, subTaskWg sync.WaitGroup

	// Channels.
	stChan := make(chan *subTask)               // Send sub-task to workers.
	rtChan := make(chan *subTask)               // Receive new sub-task from workers.
	seChan := make(chan struct{})               // Send exit signal to workers.
	reChan := make(chan struct{}, workerNumber) // Receive exit quest from workers.
	dChan := make(chan struct{})                // Receive done signal from worker manager.
	dwChan := make(chan struct{})               // For worker manager to send done signal to workers.

	// Channel for debug statistical analysis.
	debugSaChan := make(chan struct{})

	// Channels used in this goroutine.
	var subTaskOutChan chan<- *subTask = stChan
	var subTaskInChan <-chan *subTask = rtChan
	var exitOutChan chan<- struct{} = seChan
	var exitInChan <-chan struct{} = reChan
	var doneChan <-chan struct{} = dChan

	// Defer to close write channels and wait for workers to exit.
	// Before workers start, for safety.
	defer func() {
		close(exitOutChan)
		if subTaskOutChan == nil {
			subTaskOutChan = stChan
		}
		close(subTaskOutChan)
		// Remove unsent sub-task, to avoid worker manager wait forever.
		unsentNumber := pq.Len()
		if unsentNumber > 0 {
			// fmt.Fprintln(os.Stderr, "main -", unsentNumber, "sub-tasks are unsent and removed.")
			subTaskWg.Add(-unsentNumber)
		}
		// Wait for workers and worker manager exit.
		<-doneChan
		// Wait for analyser exit.
		<-debugSaChan
	}()

	// Start workers and worker manager.
	for i := 0; i < workerNumber; i++ {
		runningWg.Add(1)
		go workerProc(handler, workerSendErrTimeout, &runningWg, &subTaskWg,
			stChan, rtChan, workerErrChan, dwChan, seChan, reChan,
			&activeCounter, &handlingCounter)
	}
	subTaskWg.Add(1) // One sub-task at the beginning, the root. Other sub-tasks are created by workers, not mainProc.
	go workerManagerProc(&runningWg, &subTaskWg, dwChan, rtChan, reChan, dChan)

	// For debug
	// Statistically analyze activeCounter and handlingCounter.
	go func() {
		defer close(debugSaChan)
		var ac, hc int32
		var maxAc, maxHc int32
		var sumAc, sumHc int64
		var avgAc, avgHc float64
		var n int64
		doesContinue := true
		for doesContinue {
			select {
			case <-seChan:
				doesContinue = false
			default:
				n += 1
				ac = atomic.LoadInt32(&activeCounter)
				hc = atomic.LoadInt32(&handlingCounter)
				if ac > maxAc {
					maxAc = ac
				}
				if hc > maxHc {
					maxHc = hc
				}
				sumAc += int64(ac)
				sumHc += int64(hc)
			}
		}
		avgAc = float64(sumAc) / float64(n)
		avgHc = float64(sumHc) / float64(n)
		outFile, e := os.OpenFile("D:\\bm.txt", os.O_APPEND|os.O_WRONLY, 0600)
		if e != nil {
			outFile = os.Stderr
		} else {
			defer outFile.Close() // Ignore error.
		}
		fmt.Fprintln(outFile, "----------------------")
		fmt.Fprintln(outFile, "workerNumber =", workerNumber)
		fmt.Fprintln(outFile, "n =", n)
		fmt.Fprintln(outFile, "\t\tAC\t\tHC")
		fmt.Fprintln(outFile, "max\t\t", maxAc, "\t\t", maxHc)
		fmt.Fprintln(outFile, "avg\t\t", avgAc, "\t", avgHc)
		fmt.Fprintln(outFile, "----------------------")
	}()

	// Some variables used in loop.
	var top gocontainer.Comparable
	var ok bool
	var st *subTask
	var lastLen int
	getTop := func() {
		top, err = pq.Top()
		if err != nil {
			return
		}
		if top == nil {
			// It should never happen, but just for safety.
			err = errors.New("gotfp: runtime: top of queue is nil")
		}
		topSubTask, ok = top.(*subTask)
		if !ok {
			// It should never happen, but just for safety.
			err = gocontainer.ErrWrongType
		}
	}

	doesContinue := true
	for doesContinue {
		lastLen = pq.Len()
		select {
		case <-exitInChan: // Worker asks to exit.
			doesContinue = false
		case st, ok = <-subTaskInChan: // Receive sub-task from worker.
			if !ok {
				// Workers already exited.
				doesContinue = false
				break
			}
			// fmt.Fprintln(os.Stderr, "main - Received sub-task:", st.fileInfo.path)
			subTaskOutChan = stChan // Enable subTaskOutChan.
			err = pq.Enqueue(st)
			if err != nil {
				if pq.Len() == lastLen {
					// pq.Len() should increase but not.
					// Adjust sub-task counting.
					subTaskWg.Add(-1)
				}
				break
			}
			getTop()
		case subTaskOutChan <- topSubTask: // After sending sub-task to worker.
			// fmt.Fprintln(os.Stderr, "main - Sent sub-task:", topSubTask.fileInfo.path)
			_, err = pq.Dequeue()
			if err != nil {
				if pq.Len() == lastLen {
					// pq.Len() should decrease but not.
					// Adjust sub-task counting.
					subTaskWg.Add(1)
				}
				break
			}
			if pq.Len() > 0 {
				getTop()
			} else {
				subTaskOutChan = nil // Disable subTaskOutChan.
				topSubTask = nil
			}
		}
		if err != nil {
			doesContinue = false
		}
	}
	// fmt.Fprintln(os.Stderr, "main - End main for loop")

	return err
}

func workerProc(handler workerHandler, sendErrTimeout time.Duration,
	runningWg, subTaskWg *sync.WaitGroup, subTaskInChan <-chan *subTask,
	subTaskOutChan chan<- *subTask, errChan chan<- error,
	doneChan <-chan struct{}, exitInChan <-chan struct{},
	exitOutChan chan<- struct{}, activeCounter, handlingCounter *int32) {
	defer runningWg.Done()
	// fmt.Fprintln(os.Stderr, "worker - Start")
	atomic.AddInt32(activeCounter, 1)
	defer atomic.AddInt32(activeCounter, -1)
	var timer *time.Timer
	var timeoutChan <-chan time.Time
	if sendErrTimeout > 0 {
		timer = time.NewTimer(sendErrTimeout)
		timeoutChan = timer.C
		// Just create a timer, stop now.
		timer.Stop()
	}
	var st, newSt *subTask
	var ok, doesExit, isTimeout bool
	var err error
	var errBuf []error
	var nextFiles []*FInfo
	var info *FInfo
	var nextFilesLen, sentCount, unsentCount int
	doesContinue := true
	for doesContinue {
		atomic.AddInt32(activeCounter, -1)
		select {
		case <-exitInChan:
			atomic.AddInt32(activeCounter, 1)
			doesContinue = false
		case <-doneChan:
			atomic.AddInt32(activeCounter, 1)
			doesContinue = false
		case st, ok = <-subTaskInChan:
			atomic.AddInt32(activeCounter, 1)
			if !ok {
				doesContinue = false
				break
			}
			func() {
				defer subTaskWg.Done()
				errBuf = nil
				if st != nil {
					err = gorecover.Recover(func() {
						atomic.AddInt32(handlingCounter, 1)
						nextFiles, doesExit = handler(st, &errBuf)
						atomic.AddInt32(handlingCounter, -1)
						if doesExit {
							exitOutChan <- struct{}{}
							return
						}
						nextFilesLen = len(nextFiles)
						// fmt.Fprintln(os.Stderr, "worker -", st.fileInfo.path, "; len(nextFiles) =", nextFilesLen)
						if nextFilesLen == 0 {
							return
						}
						sentCount = 0
						subTaskWg.Add(nextFilesLen)
						defer func() {
							unsentCount = nextFilesLen - sentCount
							if unsentCount > 0 {
								subTaskWg.Add(-unsentCount) // Adjust sub-task counting.
							}
						}()
						for _, info = range nextFiles {
							if info == nil {
								continue
							}
							// fmt.Fprintln(os.Stderr, "worker - Sending new sub-task", info)
							newSt = &subTask{
								fileInfo: *info,
								depth:    st.depth + 1,
							}
							atomic.AddInt32(activeCounter, -1)
							select {
							case <-exitInChan:
								atomic.AddInt32(activeCounter, 1)
								doesContinue = false
								return
							case <-doneChan:
								atomic.AddInt32(activeCounter, 1)
								doesContinue = false
								return
							case subTaskOutChan <- newSt:
								atomic.AddInt32(activeCounter, 1)
								// fmt.Fprintln(os.Stderr, "worker - Sent new sub-task:", newSt.fileInfo.path)
								sentCount += 1
							}
						}
					})
				} else {
					err = ErrNilSubTask
				}
				if errChan != nil {
					if err != nil {
						errBuf = append(errBuf, err)
					}
					isTimeout = false
					if timer != nil && len(errBuf) > 0 {
						resetTimer(timer, sendErrTimeout)
					}
					for _, e := range errBuf {
						if isTimeout {
							select {
							case errChan <- e: // Try to send error immediately.
							default:
							}
						} else {
							atomic.AddInt32(activeCounter, -1)
							select {
							case errChan <- e:
								atomic.AddInt32(activeCounter, 1)
							case <-exitInChan:
								atomic.AddInt32(activeCounter, 1)
								doesContinue = false
								isTimeout = true
							case <-doneChan:
								atomic.AddInt32(activeCounter, 1)
								doesContinue = false
								isTimeout = true
							case <-timeoutChan:
								atomic.AddInt32(activeCounter, 1)
								isTimeout = true
							}
						}
					}
					if timer != nil {
						timer.Stop()
					}
				}
			}()
		}
	}
	// fmt.Fprintln(os.Stderr, "worker - End main for loop")
}

func workerManagerProc(runningWg, subTaskWg *sync.WaitGroup,
	doneToWorkerChan chan<- struct{}, subTaskChan chan<- *subTask,
	exitChan chan<- struct{}, doneChan chan<- struct{}) {
	defer close(doneChan)
	// fmt.Fprintln(os.Stderr, "worker manager - Start and wait for subTaskingWg")
	subTaskWg.Wait()
	close(doneToWorkerChan)
	// fmt.Fprintln(os.Stderr, "worker manager - Wait for runningWg")
	runningWg.Wait()
	// Close all write channels used by workers.
	// Not necessary, but for safety, maybe...
	close(subTaskChan)
	close(exitChan)
}
