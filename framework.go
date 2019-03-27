package gotfp

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/donyori/gocontainer"
	"github.com/donyori/gocontainer/pqueue"
	"github.com/donyori/gorecover"
)

// Note that: fmt.Println() and fmt.Printf() are for debug.

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
			fmt.Fprintln(os.Stderr, "gotfp: WARNING:", unsentNumber,
				"sub-tasks are unsent and removed.")
			subTaskWg.Add(-unsentNumber)
		}
		// Wait for workers and worker manager exit.
		<-doneChan
	}()

	// Start workers and worker manager.
	for i := 0; i < workerNumber; i++ {
		runningWg.Add(1)
		go workerProc(handler, workerSendErrTimeout, &runningWg, &subTaskWg,
			stChan, rtChan, workerErrChan, dwChan, seChan, reChan)
	}
	subTaskWg.Add(1) // One sub-task at the beginning, the root. Other sub-tasks are created by workers, not mainProc.
	go workerManagerProc(&runningWg, &subTaskWg, dwChan, rtChan,
		workerErrChan, reChan, dChan)

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
			// fmt.Println("main - Received sub-task:", st.fileInfo.path)
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
			// fmt.Println("main - Sent sub-task:", topSubTask.fileInfo.path)
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
	// fmt.Println("main - End main for loop")

	return err
}

func workerProc(handler workerHandler, sendErrTimeout time.Duration,
	runningWg, subTaskWg *sync.WaitGroup, subTaskInChan <-chan *subTask,
	subTaskOutChan chan<- *subTask, errChan chan<- error,
	doneChan <-chan struct{}, exitInChan <-chan struct{},
	exitOutChan chan<- struct{}) {
	defer runningWg.Done()
	// fmt.Println("worker - Start")
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
	doesContinue := true
	for doesContinue {
		select {
		case <-exitInChan:
			doesContinue = false
		case <-doneChan:
			doesContinue = false
		case st, ok = <-subTaskInChan:
			if !ok {
				doesContinue = false
				break
			}
			func() {
				defer subTaskWg.Done()
				errBuf = nil
				if st != nil {
					err = gorecover.Recover(func() {
						nextFiles, doesExit = handler(st, &errBuf)
						if doesExit {
							exitOutChan <- struct{}{}
							return
						}
						// fmt.Println("worker -", st.fileInfo.path, "; len(nextFiles) =", len(nextFiles))
						for _, info = range nextFiles {
							if info == nil {
								continue
							}
							// fmt.Println("worker - Sending new sub-task", info)
							newSt = &subTask{
								fileInfo: *info,
								depth:    st.depth + 1,
							}
							select {
							case <-exitInChan:
								doesContinue = false
								return
							case <-doneChan:
								doesContinue = false
								return
							case subTaskOutChan <- newSt:
								// fmt.Println("worker - Sent new sub-task:", newSt.fileInfo.path)
								subTaskWg.Add(1)
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
							select {
							case errChan <- e:
							case <-exitInChan:
								doesContinue = false
								isTimeout = true
							case <-doneChan:
								doesContinue = false
								isTimeout = true
							case <-timeoutChan:
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
	// fmt.Println("worker - End main for loop")
}

func workerManagerProc(runningWg, subTaskWg *sync.WaitGroup,
	doneToWorkerChan chan<- struct{}, subTaskChan chan<- *subTask,
	errChan chan<- error, exitChan chan<- struct{}, doneChan chan<- struct{}) {
	defer close(doneChan)
	// fmt.Println("worker manager - Start and wait for subTaskingWg")
	subTaskWg.Wait() // WARNING: Here maybe have a bug, which randomly occurs and cause a panic: sync: WaitGroup is reused before previous Wait has returned.
	close(doneToWorkerChan)
	// fmt.Println("worker manager - Wait for runningWg")
	runningWg.Wait()
	// Close all write channels used by workers.
	// Not necessary, but for safety, maybe...
	close(subTaskChan)
	if errChan != nil {
		close(errChan)
	}
	close(exitChan)
}
