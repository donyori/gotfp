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
	Others   []FInfo
	Errs     []FInfo
}

type FileHandler func(info FInfo, depth int) Action

type BatchHandler func(batch Batch, depth int) (
	action Action, skipDirs map[string]bool)
