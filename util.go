package gotfp

import (
	"os"
	"sort"
)

func readDirNames(dirPath string) (dirNames []string, err error) {
	dirFile, err := os.Open(dirPath)
	if err != nil {
		return nil, err
	}
	defer dirFile.Close() // Ignore error.
	dirNames, err = dirFile.Readdirnames(0)
	if err != nil {
		dirNames = nil
	}
	if len(dirNames) > 0 {
		sort.Strings(dirNames)
	}
	return
}
