package gotfp

import (
	"time"

	"github.com/donyori/gocommfw"
	"github.com/donyori/gocommfw/dfw"
	"github.com/donyori/gocommfw/prefab"
)

// Ensure handler != nil && workerNumber > 0 && len(roots) > 0.
func callDfw(handler taskHandler,
	workerNumber int,
	workerErrChan chan<- error,
	workerSendErrTimeout time.Duration,
	roots ...string) {
	its := make([]interface{}, 0, len(roots))
	for _, root := range roots {
		its = append(its, &tTask{
			fInfo: FInfo{Path: root},
			depth: 0,
		})
	}
	h := func(task interface{}, errBuf *[]error) (
		newTasks []interface{}, doesExit bool) {
		t := task.(*tTask)
		nextFiles, doesExit := handler(t, errBuf)
		if doesExit || len(nextFiles) == 0 {
			return nil, doesExit
		}
		newTasks = make([]interface{}, 0, len(nextFiles))
		newDepth := t.depth + 1
		for i := range nextFiles {
			newT := &tTask{
				fInfo: nextFiles[i],
				depth: newDepth,
			}
			newTasks = append(newTasks, newT)
		}
		return newTasks, false
	}
	dfw.DoEx(prefab.LdgbTaskManagerMaker, h, gocommfw.WorkerSettings{
		Number:         int32(workerNumber),
		SendErrTimeout: workerSendErrTimeout,
	}, workerErrChan, its...)
}
