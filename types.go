package gotfp

import "os"

type FInfo struct {
	Path string
	Info os.FileInfo
	Err  error
}

type Batch struct {
	Parent   FInfo
	Dirs     []FInfo
	RegFiles []FInfo
	Symlinks []FInfo
	Errs     []FInfo
}

type Action int8

type FileHandler func(info FInfo, depth int) Action

type BatchHandler func(batch Batch, depth int) (
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

func (a Action) String() string {
	if a < ActionContinue || a > ActionSkipDir {
		return "Unknown"
	}
	return actionStrings[a]
}
