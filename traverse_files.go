package gotfp

import (
	"errors"
	"os"
	"path/filepath"

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
		var dirNames []string
		path := task.FileInfo.Path
		info := task.FileInfo.Info
		err := task.FileInfo.Err
		if err == nil {
			if info == nil {
				// Didn't get file stat. Get it now.
				info, err = os.Lstat(path)
			}
			if err == nil && info != nil && info.IsDir() &&
				(info.Mode()&os.ModeSymlink) == 0 {
				// Get the name of files under this directory.
				dirNames, err = readDirNames(path)
			}
		}
		task.FileInfo.Info = info
		task.FileInfo.Err = err
		action := fileHandler(task.FileInfo, task.Depth)
		switch action {
		case ActionContinue:
			// Do nothing here.
		case ActionExit:
			return nil, true
		case ActionSkipDir:
			if info == nil || !info.IsDir() {
				*errBuf = append(*errBuf, ErrNoDirToSkip)
			}
			return
		default:
			*errBuf = append(*errBuf, NewUnknownActionError(action))
		}
		if len(dirNames) == 0 {
			return
		}
		newTasks = make([]*tTask, 0, len(dirNames))
		for i := range dirNames {
			nextPath := filepath.Join(path, dirNames[i])
			info, err = os.Lstat(nextPath)
			newTasks = append(newTasks, &tTask{FileInfo: FileInfo{
				Path: nextPath,
				Info: info,
				Err:  err,
			}})
		}
		return
	} // End of func h.
	return h
}
