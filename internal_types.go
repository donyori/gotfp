package gotfp

// Traversing task.
// One task for one file.
type tTask struct {
	fInfo FInfo
	depth int
}

// There should be no nil *FInfo in "nextFiles"!
type taskHandler func(task *tTask, errBuf *[]error) (
	nextFiles []FInfo, doesExit bool)
