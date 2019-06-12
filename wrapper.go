package gotfp

import (
	"path/filepath"

	"github.com/donyori/goctpf"
	"github.com/donyori/goctpf/idtpf/dfw"
	"github.com/donyori/goctpf/prefab"
)

// Ensure handler != nil && len(roots) > 0.
func callDfw(handler taskHandler,
	workerSettings goctpf.WorkerSettings,
	workerErrChan chan<- error,
	roots ...string) {
	its := make([]interface{}, 0, len(roots)) // initial tasks
	for i := range roots {
		// Try to get absolute path.
		root, err := filepath.Abs(roots[i])
		if err != nil {
			root = filepath.Clean(roots[i])
		}
		its = append(its, &tTask{
			FileInfo: FileInfo{Path: root},
			Depth:    0,
		})
	}
	h := func(workerNo int, task interface{}, errBuf *[]error) (
		newTasks []interface{}, doesExit bool) {
		t := task.(*tTask)
		nextTasks, doesExit := handler(t, errBuf)
		if doesExit || len(nextTasks) == 0 {
			return nil, doesExit
		}
		newTasks = make([]interface{}, 0, len(nextTasks))
		newDepth := t.Depth + 1
		for _, newTask := range nextTasks {
			if newTask.Depth <= 0 {
				newTask.Depth = newDepth
			}
			newTasks = append(newTasks, newTask)
		}
		return newTasks, false
	}
	dfw.DoEx(prefab.LdgbTaskManagerMaker, h, nil, nil,
		workerSettings, workerErrChan, its...)
}
