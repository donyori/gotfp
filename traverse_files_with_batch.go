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
				(info.Mode()&os.ModeSymlink) == 0 {
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
			if parent != "." && parent != path {
				batch := &Batch{Parent: FileInfo{Path: parent}}
				location = &LocationBatchInfo{Batch: batch}
				info, err = os.Lstat(parent)
				if err == nil {
					batch.Parent.Info = info
					if info != nil && info.IsDir() &&
						(info.Mode()&os.ModeSymlink) == 0 {
						siblingNames, err := readDirNames(parent)
						if err != nil {
							batch.Parent.Err = err
						}
						if len(siblingNames) > 0 {
							pathBase := filepath.Base(path)
							var locationSliceId int
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
								} else if (info.Mode() & os.ModeSymlink) != 0 {
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
		childPaths := make([]string, len(dirNames))
		for i := range dirNames {
			childPath := filepath.Join(path, dirNames[i])
			childPaths[i] = childPath
			info, err = os.Lstat(childPath)
			fInfo := FileInfo{
				Path: childPath,
				Info: info,
				Err:  err,
			}
			if err != nil || info == nil {
				batch.Errs = append(batch.Errs, fInfo)
			} else if (info.Mode() & os.ModeSymlink) != 0 {
				batch.Symlinks = append(batch.Symlinks, fInfo)
			} else if info.IsDir() {
				batch.Dirs = append(batch.Dirs, fInfo)
			} else if info.Mode().IsRegular() {
				batch.RegFiles = append(batch.RegFiles, fInfo)
			} else {
				batch.Others = append(batch.Others, fInfo)
			}
		}
		var indexes [5]int
		newTasks = make([]*tTask, 0, len(childPaths))
		for _, childPath := range childPaths {
			if len(batch.Dirs) > 0 &&
				childPath == batch.Dirs[indexes[0]].Path {
				location = &LocationBatchInfo{
					Batch: batch,
					Slice: batch.Dirs,
					Index: indexes[0],
				}
				indexes[0]++
			} else if len(batch.RegFiles) > 0 &&
				childPath == batch.RegFiles[indexes[1]].Path {
				location = &LocationBatchInfo{
					Batch: batch,
					Slice: batch.RegFiles,
					Index: indexes[1],
				}
				indexes[1]++
			} else if len(batch.Symlinks) > 0 &&
				childPath == batch.Symlinks[indexes[2]].Path {
				location = &LocationBatchInfo{
					Batch: batch,
					Slice: batch.Symlinks,
					Index: indexes[2],
				}
				indexes[2]++
			} else if len(batch.Others) > 0 &&
				childPath == batch.Others[indexes[3]].Path {
				location = &LocationBatchInfo{
					Batch: batch,
					Slice: batch.Others,
					Index: indexes[3],
				}
				indexes[3]++
			} else {
				location = &LocationBatchInfo{
					Batch: batch,
					Slice: batch.Errs,
					Index: indexes[4],
				}
				indexes[4]++
			}
			newTasks = append(newTasks, &tTask{
				FileInfo: location.Slice[location.Index],
				ExInfo:   location,
			})
		}
		return
	} // End of func h.
	return h
}
