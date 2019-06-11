package gotfp

import "os"

type FileInfo struct {
	Path string
	Info os.FileInfo
	Err  error
}

type Batch struct {
	Parent   FileInfo
	Dirs     []FileInfo
	RegFiles []FileInfo
	Symlinks []FileInfo
	Others   []FileInfo
	Errs     []FileInfo
}

type FileHandler func(info FileInfo, depth int) Action

type BatchHandler func(batch Batch, depth int) (
	action Action, skipDirs map[string]bool)
