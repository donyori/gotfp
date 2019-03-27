package gotfp

import (
	"os"
	"path/filepath"
	"time"
)

func TraverseFiles(root string, handler FileHandler, workerNumber int,
	workerErrChan chan<- error, workerSendErrTimeout time.Duration) error {
	if handler == nil {
		return ErrNilHandler
	}
	wh := func(st *subTask, errBuf *[]error) (
		nextFiles []*FInfo, doesExit bool) {
		// nextFiles = nil
		// doesExit = false

		if st == nil {
			// It should never happen, but just for safety.
			*errBuf = append(*errBuf, ErrNilSubTask)
			return
		}
		var dirnames []string
		info := st.fileInfo.info
		if st.fileInfo.err == nil {
			if info == nil {
				// Didn't get file stat. Get it now.
				st.fileInfo.info, st.fileInfo.err = os.Lstat(st.fileInfo.path)
				info = st.fileInfo.info
			}
			if info != nil && info.IsDir() {
				func() {
					dir, err := os.Open(st.fileInfo.path)
					if err != nil {
						st.fileInfo.err = err
						return
					}
					defer dir.Close() // Ignore error.
					dirnames, err = dir.Readdirnames(0)
					if err != nil {
						st.fileInfo.err = err
					}
				}()
			}
		}

		action := handler(st.fileInfo.Copy(), st.depth)
		switch action {
		case ActionContinue:
			// Do nothing here.
		case ActionExit:
			return nil, true
		case ActionSkipDir:
			if info != nil && !info.IsDir() {
				*errBuf = append(*errBuf, ErrNoDirToSkip)
			}
			return
		default:
			*errBuf = append(*errBuf, ErrUnknownAction)
		}
		if len(dirnames) == 0 {
			return
		}
		nextFiles = make([]*FInfo, 0, len(dirnames))
		for _, name := range dirnames {
			path := filepath.Join(st.fileInfo.path, name)
			fileInfo, err := os.Lstat(path)
			nextFiles = append(nextFiles, &FInfo{
				path: path,
				info: fileInfo,
				err:  err,
			})
		}
		return
	}
	return mainProc(root, wh, workerNumber, workerErrChan, workerSendErrTimeout)
}
