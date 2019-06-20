package gotfp

import "os"

type FileInfo struct {
	Path  string
	Cat   FileCategory
	Info  os.FileInfo
	Chldn []string
	Err   error
}

type Batch struct {
	Parent   FileInfo
	Errs     []FileInfo
	RegFiles []FileInfo
	Others   []FileInfo
	Symlinks []FileInfo
	Dirs     []FileInfo
}

type LocationBatchInfo struct {
	Batch    *Batch
	SliceIdx int
}

type FileHandler func(info FileInfo, depth int) Action

type BatchHandler func(batch Batch, depth int) (
	action Action, skipDirs map[string]bool)

type FileWithBatchHandler func(info FileInfo, lctn *LocationBatchInfo,
	depth int) Action
