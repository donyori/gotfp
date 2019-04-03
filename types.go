package gotfp

import "os"

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

func (a Action) String() string {
	if a < ActionContinue || a > ActionSkipDir {
		return "Unknown"
	}
	return actionStrings[a]
}
