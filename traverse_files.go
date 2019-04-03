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
		// No files to traverse. Just exit.
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
		nextFiles []*FInfo, doesExit bool) {
		var dirNames []string
		info := task.fInfo.info
		err := task.fInfo.err
		if err == nil {
			if info == nil {
				// Didn't get file stat. Get it now.
				info, err = os.Lstat(task.fInfo.path)
			}
			if err == nil && info != nil && info.IsDir() {
				// Get the name of files under this directory.
				dirNames, err = readDirNames(task.fInfo.path)
			}
		}
		task.fInfo.info = info
		task.fInfo.err = err
		action := fileHandler(task.fInfo.Copy(), task.depth)
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
			*errBuf = append(*errBuf, ErrUnknownAction)
		}
		if len(dirNames) == 0 {
			return
		}
		nextFiles = make([]*FInfo, 0, len(dirNames))
		for _, name := range dirNames {
			path := filepath.Join(task.fInfo.path, name)
			fileInfo, err := os.Lstat(path)
			nextFiles = append(nextFiles, &FInfo{
				path: path,
				info: fileInfo,
				err:  err,
			})
		}
		return
	} // End of func h.
	return h
}
