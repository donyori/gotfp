package gotfp

import (
	"os"

	"github.com/donyori/gocontainer"
)

// Implement github.com/donyori/gocontainer.Comparable.
type FInfo struct {
	path string
	info os.FileInfo
	err  error
}

type Batch struct {
	Parent   FInfo
	Dirs     []FInfo
	RegFiles []FInfo
	Symlinks []FInfo
	Errs     []FInfo
}

type Action int8

type FileHandler func(info *FInfo, depth int) Action

type BatchHandler func(batch *Batch, depth int) (
	action Action, skipDirs map[string]bool)

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

func (fi *FInfo) Copy() *FInfo {
	if fi == nil {
		return nil
	}
	return &FInfo{
		path: fi.path,
		info: fi.info,
		err:  fi.err,
	}
}

func (fi *FInfo) Less(another gocontainer.Comparable) (res bool, err error) {
	a, ok := another.(*FInfo)
	if !ok {
		return false, gocontainer.ErrWrongType
	}
	if fi != nil {
		if a != nil {
			// Priority: path > info > err
			if fi.path != a.path {
				// Most case.
				return fi.path > a.path, nil // Alphabetical order.
			} else if fi.info == nil || a.info == nil {
				if fi.info == nil && a.info != nil {
					return true, nil
				}
			} else if fi.err != nil && a.err == nil {
				return true, nil
			}
		}
	} else if a != nil {
		return true, nil
	}
	return false, nil
}

func (a Action) String() string {
	if a < ActionContinue || a > ActionSkipDir {
		return "Unknown"
	}
	return actionStrings[a]
}
