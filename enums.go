package gotfp

import "strings"

type Action int8
type FileCategory int8

const (
	ActionContinue Action = iota + 1
	ActionExit
	ActionSkip
)

const (
	ErrorFile FileCategory = iota + 1
	RegularFile
	OtherFile
	Symlink
	Directory
)

var actionStrings = [...]string{
	"Unknown",
	"Continue",
	"Exit",
	"Skip",
}

var fileCategoryStrings = [...]string{
	"Unknown",
	"ErrorFile",
	"RegularFile",
	"OtherFile",
	"Symlink",
	"Directory",
}

func ParseAction(s string) Action {
	for i := range actionStrings {
		if strings.EqualFold(s, actionStrings[i]) {
			return Action(i)
		}
	}
	return 0 // Stands for "Unknown".
}

func (a Action) String() string {
	if a < ActionContinue || a > ActionSkip {
		return actionStrings[0]
	}
	return actionStrings[a]
}

func (a Action) MarshalText() ([]byte, error) {
	return []byte(a.String()), nil
}

func (a *Action) UnmarshalText(text []byte) error {
	*a = ParseAction(string(text))
	return nil
}

func ParseFileCategory(s string) FileCategory {
	for i := range fileCategoryStrings {
		if strings.EqualFold(s, fileCategoryStrings[i]) {
			return FileCategory(i)
		}
	}
	return 0 // Stands for "Unknown".
}

func (fc FileCategory) String() string {
	if fc < ErrorFile || fc > Directory {
		return fileCategoryStrings[0]
	}
	return fileCategoryStrings[fc]
}

func (fc FileCategory) MarshalText() ([]byte, error) {
	return []byte(fc.String()), nil
}

func (fc *FileCategory) UnmarshalText(text []byte) error {
	*fc = ParseFileCategory(string(text))
	return nil
}
