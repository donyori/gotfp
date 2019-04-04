package gotfp

import (
	"os"
	"path/filepath"
	"time"
)

func TraverseFiles(handler FileHandler,
	workerNumber int,
	workerErrChan chan<- error,
	workerSendErrTimeout time.Duration,
	roots ...string) error {
	if handler == nil {
		return ErrNilHandler
	}
	if workerNumber <= 0 {
		return ErrNonPositiveWorkerNumber
	}
	if len(roots) == 0 {
		// No file to traverse. Just exit.
		return nil
	}
	h := makeTraverseFilesHandler(handler)
	err := callDfw(h, workerNumber, workerErrChan,
		workerSendErrTimeout, roots...)
	return err
}

// Ensure fileHandler != nil.
func makeTraverseFilesHandler(fileHandler FileHandler) taskHandler {
	h := func(task *tTask, errBuf *[]error) (
		nextFiles []FInfo, doesExit bool) {
		var dirNames []string
		path := task.fInfo.path
		info := task.fInfo.info
		err := task.fInfo.err
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
		task.fInfo.info = info
		task.fInfo.err = err
		action := fileHandler(task.fInfo, task.depth)
		switch action {
		case ActionContinue:
			// Do nothing here.
		case ActionExit:
			return nil, true
		case ActionSkipDir:
			if info == nil || !info.IsDir() ||
				(info.Mode()&os.ModeSymlink) != 0 {
				*errBuf = append(*errBuf, ErrNoDirToSkip)
			}
			return
		default:
			*errBuf = append(*errBuf, ErrUnknownAction)
		}
		if len(dirNames) == 0 {
			return
		}
		nextFiles = make([]FInfo, 0, len(dirNames))
		for _, name := range dirNames {
			nextPath := filepath.Join(path, name)
			info, err = os.Lstat(nextPath)
			nextFiles = append(nextFiles, FInfo{
				path: nextPath,
				info: info,
				err:  err,
			})
		}
		return
	} // End of func h.
	return h
}
