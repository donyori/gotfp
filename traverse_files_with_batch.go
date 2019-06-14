package gotfp

import (
	"errors"
	"os"
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
				info.Mode()&os.ModeSymlink == 0 {
				// Get the name of files under this directory.
				dirNames, err = readDirNames(path)
			}
		}
		task.FileInfo.Info = info
		task.FileInfo.Err = err
		var location *LocationBatchInfo
		if task.ExInfo != nil {
			location = task.ExInfo.(*LocationBatchInfo)
		} else if path != "" {
			parent := filepath.Dir(path)
			if parent != path { // path is not a root file path.
				batch := &Batch{Parent: FileInfo{Path: parent}}
				location = &LocationBatchInfo{Batch: batch}
				info, err = os.Lstat(parent)
				if err == nil {
					batch.Parent.Info = info
					if info != nil && info.IsDir() &&
						info.Mode()&os.ModeSymlink == 0 {
						siblingNames, err := readDirNames(parent)
						if err != nil {
							batch.Parent.Err = err
						}
						if len(siblingNames) > 0 {
							pathBase := filepath.Base(path)
							locationSliceId := -1
							// Don't set location.Slice during the loop,
							// because it may be changed by append().
							for _, name := range siblingNames {
								var fInfo FileInfo
								if pathBase != name {
									siblingPath := filepath.Join(parent, name)
									info, err = os.Lstat(siblingPath)
									fInfo.Path = siblingPath
									fInfo.Info = info
									fInfo.Err = err
								} else {
									fInfo = task.FileInfo
									info = fInfo.Info
									err = fInfo.Err
								}
								if err != nil || info == nil {
									batch.Errs = append(batch.Errs, fInfo)
									if pathBase == name {
										locationSliceId = 4
										location.Index = len(batch.Errs) - 1
									}
								} else if info.Mode()&os.ModeSymlink != 0 {
									batch.Symlinks = append(batch.Symlinks, fInfo)
									if pathBase == name {
										locationSliceId = 2
										location.Index = len(batch.Symlinks) - 1
									}
								} else if info.IsDir() {
									batch.Dirs = append(batch.Dirs, fInfo)
									if pathBase == name {
										locationSliceId = 0
										location.Index = len(batch.Dirs) - 1
									}
								} else if info.Mode().IsRegular() {
									batch.RegFiles = append(batch.RegFiles, fInfo)
									if pathBase == name {
										locationSliceId = 1
										location.Index = len(batch.RegFiles) - 1
									}
								} else {
									batch.Others = append(batch.Others, fInfo)
									if pathBase == name {
										locationSliceId = 3
										location.Index = len(batch.Others) - 1
									}
								}
							} // End of for loop.
							switch locationSliceId {
							case 0:
								location.Slice = batch.Dirs
							case 1:
								location.Slice = batch.RegFiles
							case 2:
								location.Slice = batch.Symlinks
							case 3:
								location.Slice = batch.Others
							case 4:
								location.Slice = batch.Errs
							}
						}
					}
				} else {
					batch.Parent.Err = err
				}
				info = task.FileInfo.Info
			}
		}
		action := fileWithBatchHandler(task.FileInfo, location, task.Depth)
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
		batch := &Batch{Parent: task.FileInfo}
		for i := range dirNames {
			childPath := filepath.Join(path, dirNames[i])
			info, err = os.Lstat(childPath)
			fInfo := FileInfo{
				Path: childPath,
				Info: info,
				Err:  err,
			}
			if err != nil || info == nil {
				batch.Errs = append(batch.Errs, fInfo)
			} else if info.Mode()&os.ModeSymlink != 0 {
				batch.Symlinks = append(batch.Symlinks, fInfo)
			} else if info.IsDir() {
				batch.Dirs = append(batch.Dirs, fInfo)
			} else if info.Mode().IsRegular() {
				batch.RegFiles = append(batch.RegFiles, fInfo)
			} else {
				batch.Others = append(batch.Others, fInfo)
			}
		}
		newTasks = make([]*tTask, 0, len(dirNames))
		slices := [...][]FileInfo{batch.Errs, batch.RegFiles,
			batch.Others, batch.Symlinks, batch.Dirs}
		for _, slice := range slices {
			for i := range slice {
				location = &LocationBatchInfo{
					Batch: batch,
					Slice: slice,
					Index: i,
				}
				newTasks = append(newTasks, &tTask{
					FileInfo: slice[i],
					ExInfo:   location,
				})
			}
		}
		return
	} // End of func h.
	return h
}
