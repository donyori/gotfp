package gotfp

import (
	"os"
	"path/filepath"
	"time"
)

func TraverseBatches(handler BatchHandler,
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
		// No batch to traverse. Just exit.
		return nil
	}
	h := makeTraverseBatchesHandler(handler)
	err := callDfw(h, workerNumber, workerErrChan,
		workerSendErrTimeout, roots...)
	return err
}

// Ensure batchHandler != nil.
func makeTraverseBatchesHandler(batchHandler BatchHandler) taskHandler {
	h := func(task *tTask, errBuf *[]error) (
		nextFiles []FInfo, doesExit bool) {
		var dirNames []string
		var batch Batch
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
		batch.Parent = task.fInfo
		if len(dirNames) > 0 {
			for _, name := range dirNames {
				dirPath := filepath.Join(path, name)
				info, err = os.Lstat(dirPath)
				fInfo := FInfo{
					path: dirPath,
					info: info,
					err:  err,
				}
				if err != nil || info == nil {
					batch.Errs = append(batch.Errs, fInfo)
				} else if (info.Mode() & os.ModeSymlink) != 0 {
					batch.Symlinks = append(batch.Symlinks, fInfo)
				} else if info.IsDir() {
					batch.Dirs = append(batch.Dirs, fInfo)
				} else {
					batch.RegFiles = append(batch.RegFiles, fInfo)
				}
			}
			info = task.fInfo.info
		}
		// Copy batch.Dirs. See https://github.com/go101/go101/wiki for details.
		dirs := append(batch.Dirs[:0:0], batch.Dirs...)
		action, skipDirs := batchHandler(batch, task.depth)
		switch action {
		case ActionContinue:
			// Do nothing here.
			// skipDirs will be ignored.
		case ActionExit:
			return nil, true
		case ActionSkipDir:
			if info == nil || !info.IsDir() ||
				(info.Mode()&os.ModeSymlink) != 0 {
				*errBuf = append(*errBuf, ErrNoDirToSkip)
				return
			}
			if len(skipDirs) == 0 {
				// Skip all sub-directories.
				return
			}
			var i, j int
			for n := len(dirs); i < n; i++ {
				if skipDirs[dirs[i].path] || skipDirs[dirs[i].info.Name()] {
					continue
				}
				if i != j {
					dirs[j] = dirs[i]
				}
				j++
			}
			nextFiles = dirs[:j]
			if i == j {
				*errBuf = append(*errBuf, ErrNoDirToSkip)
			}
			return
		default:
			*errBuf = append(*errBuf, ErrUnknownAction)
		}
		nextFiles = dirs
		return
	} // End of func h.
	return h
}