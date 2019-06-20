package gotfp

import (
	"errors"
	"fmt"
)

type UnknownActionError struct {
	action interface{}
}

type UnknownFileCategoryError struct {
	fileCategory interface{}
}

var ErrNoDirToSkip error = errors.New("gotfp: no directory to skip")

func NewUnknownActionError(action interface{}) error {
	switch action.(type) {
	case Action:
		a := action.(Action)
		if a >= ActionContinue && a <= ActionSkipDir {
			panic(fmt.Errorf("gotfp: action %q is known but mark as unknown", a))
		}
	case string:
		// Do nothing.
	default:
		panic(fmt.Errorf(
			"gotfp: type of action should be Action or string, but got %T",
			action))
	}
	return &UnknownActionError{action: action}
}

func (uae *UnknownActionError) Error() string {
	switch uae.action.(type) {
	case Action:
		return fmt.Sprintf("gotfp: action (%d) is unknown", uae.action)
	default:
		return fmt.Sprintf("gotfp: action (%s) is unknown", uae.action)
	}
}

func NewUnknownFileCategoryError(fileCategory interface{}) error {
	switch fileCategory.(type) {
	case FileCategory:
		fc := fileCategory.(FileCategory)
		if fc >= ErrorFile && fc <= Directory {
			panic(fmt.Errorf(
				"gotfp: file category %q is known but mark as unknown", fc))
		}
	case string:
		// Do nothing.
	default:
		panic(fmt.Errorf(
			"gotfp: type of fileCategory should be FileCategory or string, but got %T",
			fileCategory))
	}
	return &UnknownFileCategoryError{fileCategory: fileCategory}
}

func (ufce *UnknownFileCategoryError) Error() string {
	switch ufce.fileCategory.(type) {
	case FileCategory:
		return fmt.Sprintf("gotfp: file category (%d) is unknown",
			ufce.fileCategory)
	default:
		return fmt.Sprintf("gotfp: file category (%s) is unknown",
			ufce.fileCategory)
	}
}
