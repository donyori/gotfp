package gotfp

// Traversing task.
// One task for one file.
type tTask struct {
	FileInfo FileInfo
	Depth    int
	ExInfo   interface{}
}

// There should be no nil *FInfo in "nextFiles"!
type taskHandler func(task *tTask, errBuf *[]error) (
	newTasks []*tTask, doesExit bool)
