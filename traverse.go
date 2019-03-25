package gotfp

import (
	"sync"

	"github.com/donyori/gocontainer"
	"github.com/donyori/gocontainer/pqueue"
)

func TraverseFiles(root string, handler FileHandler,
	workerNumber int, workerErrChan chan<- error) error {
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

	var runningWg, busyWg sync.WaitGroup
	chanE := make(chan struct{})                 // Ask worker and reporter to exit.
	chanSW := make(chan *subTask)                // Send sub-task to worker.
	chanRW := make(chan *subTask)                // Receive new sub-task from worker.
	chanREW := make(chan struct{}, workerNumber) // Receive exit quest from worker.
	chanRR := make(chan struct{}, 1)             // Receive report from reporter.
	chanCR := make(chan struct{}, 1)             // Ask reporter to continue.
	for i := 0; i < workerNumber; i++ {
		runningWg.Add(1)
		busyWg.Add(1) // Set busy at the beginning, to avoid idle report.
		go traverseFilesWorker(&runningWg, &busyWg, chanSW, chanRW,
			workerErrChan, chanE, chanREW)
	}
	go closeWorkerWriteChans(&runningWg, chanRW, workerErrChan, chanREW)
	runningWg.Add(1)
	go workersAllIdleReporter(&runningWg, &busyWg, chanRR, chanCR, chanE)

	var outChan chan<- *subTask = chanSW
	var inChan <-chan *subTask = chanRW
	var exitOutChan chan<- struct{} = chanE
	var exitInChan <-chan struct{} = chanREW
	var idleChan <-chan struct{} = chanRR
	var continueChan chan<- struct{} = chanCR
	defer func() {
		if exitOutChan != nil {
			close(exitOutChan)
			exitOutChan = nil
		}
		if continueChan != nil {
			close(continueChan)
			continueChan = nil
		}
		if outChan == nil {
			outChan = chanSW
		}
		close(outChan)
		outChan = nil
		runningWg.Wait()
	}()

	// Set first sub-task, the root.
	topSubTask := &subTask{path: root, depth: 0}
	var top gocontainer.Comparable
	var ok bool
	var st *subTask
	getTop := func() {
		top, err = pq.Top()
		if err != nil {
			return
		}
		topSubTask, ok = top.(*subTask)
		if !ok {
			err = gocontainer.ErrWrongType
		}
	}

	doesContinue := true
	for doesContinue {
		select {
		case <-exitInChan:
			doesContinue = false
		case st, ok = <-inChan:
			if !ok {
				doesContinue = false
				break
			}
			outChan = chanSW // Enable outChan.
			err = pq.Enqueue(st)
			if err != nil {
				break
			}
			getTop()
		case outChan <- topSubTask:
			// Firstly, check whether pq is empty or not.
			if pq.Len() > 0 {
				outChan = chanSW // Enable outChan.
				_, err = pq.Dequeue()
				if err != nil {
					break
				}
				getTop()
			} else {
				outChan = nil // Disable outChan.
				topSubTask = nil
			}
		case <-idleChan:
			if pq.Len() > 0 || topSubTask != nil {
				continueChan <- struct{}{}
			} else {
				doesContinue = false
			}
		}
		if err != nil {
			doesContinue = false
		}
	}

	return err
}

func traverseFilesWorker(runningWg, busyWg *sync.WaitGroup,
	inChan <-chan *subTask, outChan chan<- *subTask, errChan chan<- error,
	exitInChan <-chan struct{}, exitOutChan chan<- struct{}) {
	defer runningWg.Done()
}

func closeWorkerWriteChans(runningWg *sync.WaitGroup,
	c1 chan<- *subTask, c2 chan<- error, c3 chan<- struct{}) {
	runningWg.Wait()
	if c1 != nil {
		close(c1)
	}
	if c2 != nil {
		close(c2)
	}
	if c3 != nil {
		close(c3)
	}
}

func workersAllIdleReporter(runningWg, busyWg *sync.WaitGroup,
	outChan chan<- struct{}, continueChan <-chan struct{},
	exitChan <-chan struct{}) {
	defer close(outChan)
	defer runningWg.Done()
	doesContinue := true
	for doesContinue {
		busyWg.Wait()
		outChan <- struct{}{}
		select {
		case <-exitChan:
			doesContinue = false
		case _, ok := <-continueChan:
			if !ok {
				doesContinue = false
			}
			// Else, just continue.
		}
	}
}
