package gotfp

import "errors"

var (
	ErrNoDirToSkip   error = errors.New("gotfp: no directory to skip")
	ErrUnknownAction error = errors.New("gotfp: action is unknown")
)
