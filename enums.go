package gotfp

import (
	"encoding/json"
	"strings"
)

type Action int8

const (
	ActionContinue Action = iota
	ActionExit
	ActionSkipDir
)

var actionStrings = [...]string{
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
	return -1 // Stands for "Unknown".
}

func (a Action) String() string {
	if a < ActionContinue || a > ActionSkipDir {
		return "Unknown"
	}
	return actionStrings[a]
}

func (a Action) MarshalJSON() ([]byte, error) {
	s := a.String()
	return json.Marshal(s)
}

func (a *Action) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	*a = ParseAction(s)
	return nil
}
