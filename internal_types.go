package gotfp

// Traversing task.
// One task for one file.
type tTask struct {
	fileInfo FileInfo
	depth    int
}

// There should be no nil *FInfo in "nextFiles"!
type taskHandler func(task *tTask, errBuf *[]error) (
	nextFiles []FileInfo, doesExit bool)
