package gotfp

import (
	"os"

	"github.com/donyori/gocontainer"
)

type FInfo struct {
	path string
	info os.FileInfo
}

type Batch struct {
	Parent   FInfo
	Dirs     []FInfo
	RegFiles []FInfo
	Symlinks []FInfo
}

type Action int8

type FileHandler func(info *FInfo, traverseErr error) (action Action, err error)

type BatchHandler func(batch *Batch, traverseErr error) (
	action Action, skipDirs []string, err error)

// For traverse.
// Implement github.com/donyori/gocontainer.Comparable.
type subTask struct {
	path  string
	depth int
}

const (
	ActionContinue Action = iota
	ActionExit
	ActionSkipDir
)

var actionStrings = [...]string{
	"Continue",
	"Exit",
	"SkipDir",
}

func (a Action) String() string {
	if a < ActionContinue || a > ActionSkipDir {
		return "Unknown"
	}
	return actionStrings[a]
}

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
	// Priority: depth > path
	if st.depth != a.depth {
		return st.depth < a.depth, nil
	} else {
		return st.path > a.path, nil // Alphabetical order.
	}
}
