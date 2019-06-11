package gotfp

import (
	"github.com/donyori/goctpf"
	"github.com/donyori/goctpf/idtpf/dfw"
	"github.com/donyori/goctpf/prefab"
)

// Ensure handler != nil && len(roots) > 0.
func callDfw(handler taskHandler,
	workerSettings goctpf.WorkerSettings,
	workerErrChan chan<- error,
	roots ...string) {
	its := make([]interface{}, 0, len(roots))
	for _, root := range roots {
		its = append(its, &tTask{
			fileInfo: FileInfo{Path: root},
			depth:    0,
		})
	}
	h := func(workerNo int, task interface{}, errBuf *[]error) (
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
				fileInfo: nextFiles[i],
				depth:    newDepth,
			}
			newTasks = append(newTasks, newT)
		}
		return newTasks, false
	}
	dfw.DoEx(prefab.LdgbTaskManagerMaker, h, nil, nil,
		workerSettings, workerErrChan, its...)
}
