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

type LocationBatchInfo struct {
	Batch *Batch
	Slice []FileInfo
	Index int
}

type FileHandler func(info FileInfo, depth int) Action

type BatchHandler func(batch Batch, depth int) (
	action Action, skipDirs map[string]bool)

type FileWithBatchHandler func(info FileInfo, location *LocationBatchInfo,
	depth int) Action
