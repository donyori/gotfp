package gotfp

import "github.com/donyori/gocontainer"

// Implement github.com/donyori/gocontainer.Comparable.
// One sub-task stands for one file.
type subTask struct {
	fileInfo FInfo
	depth    int
}

type workerHandler func(st *subTask, errBuf *[]error) (
	nextFiles []*FInfo, doesExit bool)

func (st *subTask) Less(another gocontainer.Comparable) (res bool, err error) {
	if st == nil {
		return false, ErrNilSubTask
	}
	a, ok := another.(*subTask)
	if !ok {
		return false, gocontainer.ErrWrongType
	}
	if a == nil {
		return false, ErrNilSubTask
	}
	// Priority: depth > fileInfo
	if st.depth != a.depth {
		return st.depth < a.depth, nil
	}
	return st.fileInfo.Less(&(a.fileInfo))
}
