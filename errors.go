package gotfp

import "errors"

var (
	ErrNilHandler              error = errors.New("gotfp: handler is nil")
	ErrNonPositiveWorkerNumber error = errors.New("gotfp: worker number is non-positive")
	ErrNilSubTask              error = errors.New("gotfp: sub-task is nil")
	ErrNoDirToSkip             error = errors.New("gotfp: no directory to skip")
	ErrUnknownAction           error = errors.New("gotfp: action is unknown")
)
