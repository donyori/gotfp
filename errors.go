package gotfp

import (
	"errors"
	"fmt"
)

type UnknownActionError struct {
	action interface{}
}

var ErrNoDirToSkip error = errors.New("gotfp: no directory to skip")

func NewUnknownActionError(action interface{}) error {
	switch action.(type) {
	case Action:
	case string:
	default:
		panic(fmt.Errorf(
			"gotfp: type of action should be Action or string, but got %T",
			action))
	}
	return &UnknownActionError{action: action}
}

func (uae *UnknownActionError) Error() string {
	return fmt.Sprintf("gotfp: action (%v) is unknown", uae.action)
}
