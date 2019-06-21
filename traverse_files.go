package gotfp

import (
	"errors"
	"path/filepath"
	"sort"

	"github.com/donyori/goctpf"
)

func TraverseFiles(handler FileHandler,
	workerSettings goctpf.WorkerSettings,
	workerErrChan chan<- error,
	roots ...string) {
	if handler == nil {
		panic(errors.New("gotfp: file handler is nil"))
	}
	if len(roots) == 0 {
		// No file to traverse. Just exit.
		return
	}
	h := makeTraverseFilesHandler(handler)
	callDfw(h, workerSettings, workerErrChan, roots...)
}

// Ensure fileHandler != nil.
func makeTraverseFilesHandler(fileHandler FileHandler) taskHandler {
	h := func(task *tTask, errBuf *[]error) (newTasks []*tTask, doesExit bool) {
		path := task.FileInfo.Path
		if task.FileInfo.Cat == 0 {
			task.FileInfo = GetFileInfo(path)
		}
		// Copy task.FileInfo.Chldn. See https://github.com/go101/go101/wiki for details.
		chldn := append(task.FileInfo.Chldn[:0:0], task.FileInfo.Chldn...)
		action := fileHandler(task.FileInfo, task.Depth)
		switch action {
		case ActionContinue:
			// Do nothing here.
		case ActionExit:
			return nil, true
		case ActionSkipDir:
			return
		default:
			*errBuf = append(*errBuf, NewUnknownActionError(action))
		}
		if len(chldn) == 0 {
			return
		}
		newTasks = make([]*tTask, 0, len(chldn))
		for i := range chldn {
			newTasks = append(newTasks, &tTask{
				FileInfo: GetFileInfo(filepath.Join(path, chldn[i])),
			})
		}
		sort.Slice(newTasks, func(i, j int) bool {
			t1 := newTasks[i]
			t2 := newTasks[j]
			if t1.FileInfo.Cat == t2.FileInfo.Cat {
				return t1.FileInfo.Path < t2.FileInfo.Path
			}
			return t1.FileInfo.Cat < t2.FileInfo.Cat
		})
		return
	} // End of func h.
	return h
}
