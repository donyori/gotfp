package gotfp

import "strings"

type Action int8

const (
	ActionContinue Action = iota + 1
	ActionExit
	ActionSkipDir
)

var actionStrings = [...]string{
	"Unknown",
	"Continue",
	"Exit",
	"SkipDir",
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
	if a < ActionContinue || a > ActionSkipDir {
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
