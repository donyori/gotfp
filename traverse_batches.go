package gotfp

import (
	"errors"
	"path/filepath"

	"github.com/donyori/goctpf"
)

func TraverseBatches(handler BatchHandler,
	workerSettings goctpf.WorkerSettings,
	workerErrChan chan<- error,
	roots ...string) {
	if handler == nil {
		panic(errors.New("gotfp: batch handler is nil"))
	}
	if len(roots) == 0 {
		// No batch to traverse. Just exit.
		return
	}
	h := makeTraverseBatchesHandler(handler)
	callDfw(h, workerSettings, workerErrChan, roots...)
}

// Ensure batchHandler != nil.
func makeTraverseBatchesHandler(batchHandler BatchHandler) taskHandler {
	h := func(task *tTask, errBuf *[]error) (newTasks []*tTask, doesExit bool) {
		path := task.FileInfo.Path
		if task.FileInfo.Cat == 0 {
			task.FileInfo = GetFileInfo(path)
		}
		chldn := task.FileInfo.Chldn
		batch := Batch{Parent: task.FileInfo}
		for i := range chldn {
			fileInfo := GetFileInfo(filepath.Join(path, chldn[i]))
			switch fileInfo.Cat {
			case ErrorFile:
				batch.Errs = append(batch.Errs, fileInfo)
			case RegularFile:
				batch.RegFiles = append(batch.RegFiles, fileInfo)
			case OtherFile:
				batch.Others = append(batch.Others, fileInfo)
			case Symlink:
				batch.Symlinks = append(batch.Symlinks, fileInfo)
			case Directory:
				batch.Dirs = append(batch.Dirs, fileInfo)
			default:
				*errBuf = append(*errBuf,
					NewUnknownFileCategoryError(fileInfo.Cat))
			}
		}
		// Copy batch.Dirs. See https://github.com/go101/go101/wiki for details.
		dirs := append(batch.Dirs[:0:0], batch.Dirs...)
		action, skipDirs := batchHandler(batch, task.Depth)
		switch action {
		case ActionContinue:
			// Do nothing here.
			// skipDirs will be ignored.
		case ActionExit:
			return nil, true
		case ActionSkip:
			if len(skipDirs) == 0 {
				// Skip all sub-directories.
				return
			}
			var i, j int
			for n := len(dirs); i < n; i++ {
				if skipDirs[dirs[i].Path] || skipDirs[dirs[i].Info.Name()] {
					continue
				}
				if i != j {
					dirs[j] = dirs[i]
				}
				j++
			}
			if j > 0 {
				newTasks = make([]*tTask, 0, j)
				for k := 0; k < j; k++ {
					newTasks = append(newTasks, &tTask{FileInfo: dirs[k]})
				}
			}
			if i == j {
				*errBuf = append(*errBuf, ErrNoDirToSkip)
			}
			return
		default:
			*errBuf = append(*errBuf, NewUnknownActionError(action))
		}
		if len(dirs) > 0 {
			newTasks = make([]*tTask, 0, len(dirs))
			for i := range dirs {
				newTasks = append(newTasks, &tTask{FileInfo: dirs[i]})
			}
		}
		return
	} // End of func h.
	return h
}
