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
	h := func(task *tTask, errBuf *[]error) (
		nextFiles []FInfo, doesExit bool) {
		var dirNames []string
		path := task.fInfo.Path
		info := task.fInfo.Info
		err := task.fInfo.Err
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
		task.fInfo.Info = info
		task.fInfo.Err = err
		action := fileHandler(task.fInfo, task.depth)
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
		nextFiles = make([]FInfo, 0, len(dirNames))
		for _, name := range dirNames {
			nextPath := filepath.Join(path, name)
			info, err = os.Lstat(nextPath)
			nextFiles = append(nextFiles, FInfo{
				Path: nextPath,
				Info: info,
				Err:  err,
			})
		}
		return
	} // End of func h.
	return h
}
