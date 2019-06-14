package gotfp

import (
	"errors"
	"os"
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
		sort.Slice(newTasks, func(i, j int) bool {
			t1 := newTasks[i]
			t2 := newTasks[j]
			info1 := t1.FileInfo.Info
			info2 := t2.FileInfo.Info
			if t1.FileInfo.Err != nil || info1 == nil {
				return t2.FileInfo.Err == nil && info2 != nil ||
					t1.FileInfo.Path < t2.FileInfo.Path
			}
			if t2.FileInfo.Err != nil || info2 == nil {
				return false
			}
			if info1.IsDir() && info1.Mode()&os.ModeSymlink == 0 {
				return info2.IsDir() && info2.Mode()&os.ModeSymlink == 0 &&
					t1.FileInfo.Path < t2.FileInfo.Path
			}
			if info2.IsDir() && info2.Mode()&os.ModeSymlink == 0 {
				return true
			}
			if info1.Mode().IsRegular() {
				return !info2.Mode().IsRegular() ||
					t1.FileInfo.Path < t2.FileInfo.Path
			}
			if info2.Mode().IsRegular() {
				return false
			}
			if info1.Mode()&os.ModeSymlink != 0 {
				return info2.Mode()&os.ModeSymlink != 0 &&
					t1.FileInfo.Path < t2.FileInfo.Path
			}
			if info2.Mode()&os.ModeSymlink != 0 {
				return true
			}
			return t1.FileInfo.Path < t2.FileInfo.Path
		})
		return
	} // End of func h.
	return h
}
