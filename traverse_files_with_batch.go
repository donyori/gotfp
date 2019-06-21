package gotfp

import (
	"errors"
	"path/filepath"

	"github.com/donyori/goctpf"
)

func TraverseFilesWithBatch(handler FileWithBatchHandler,
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
	h := makeTraverseFilesWithBatchHandler(handler)
	callDfw(h, workerSettings, workerErrChan, roots...)
}

// Ensure fileWithBatchHandler != nil.
func makeTraverseFilesWithBatchHandler(
	fileWithBatchHandler FileWithBatchHandler) taskHandler {
	h := func(task *tTask, errBuf *[]error) (newTasks []*tTask, doesExit bool) {
		path := task.FileInfo.Path
		if task.FileInfo.Cat == 0 {
			task.FileInfo = GetFileInfo(path)
		}
		// Copy task.FileInfo.Chldn. See https://github.com/go101/go101/wiki for details.
		chldn := append(task.FileInfo.Chldn[:0:0], task.FileInfo.Chldn...)
		var lctn *LocationBatchInfo
		if task.ExInfo != nil {
			lctn = task.ExInfo.(*LocationBatchInfo)
		} else if path != "" {
			parent := filepath.Dir(path)
			if parent != path { // path is not a root file path.
				batch := &Batch{Parent: GetFileInfo(parent)}
				lctn = &LocationBatchInfo{Batch: batch}
				if len(batch.Parent.Chldn) > 0 {
					pathBase := filepath.Base(path)
					for _, name := range batch.Parent.Chldn {
						var fileInfo FileInfo
						if pathBase != name {
							fileInfo = GetFileInfo(filepath.Join(parent, name))
						} else {
							fileInfo = task.FileInfo
							switch fileInfo.Cat {
							case ErrorFile:
								lctn.SliceIdx = len(batch.Errs)
							case RegularFile:
								lctn.SliceIdx = len(batch.RegFiles)
							case OtherFile:
								lctn.SliceIdx = len(batch.Others)
							case Symlink:
								lctn.SliceIdx = len(batch.Symlinks)
							case Directory:
								lctn.SliceIdx = len(batch.Dirs)
							}
							// UnknownFileCategoryError will be reported in the following step.
						}
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
				}
			}
		}
		action := fileWithBatchHandler(task.FileInfo, lctn, task.Depth)
		switch action {
		case ActionContinue:
			// Do nothing here.
		case ActionExit:
			return nil, true
		case ActionSkip:
			return
		default:
			*errBuf = append(*errBuf, NewUnknownActionError(action))
		}
		if len(chldn) == 0 {
			return
		}
		batch := &Batch{Parent: task.FileInfo}
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
		newTasks = make([]*tTask, 0, len(chldn))
		slices := [...][]FileInfo{batch.Errs, batch.RegFiles,
			batch.Others, batch.Symlinks, batch.Dirs}
		for _, slice := range slices {
			for i := range slice {
				lctn = &LocationBatchInfo{
					Batch:    batch,
					SliceIdx: i,
				}
				newTasks = append(newTasks, &tTask{
					FileInfo: slice[i],
					ExInfo:   lctn,
				})
			}
		}
		return
	} // End of func h.
	return h
}
