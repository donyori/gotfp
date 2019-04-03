package gotfp

import "os"

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
	return
}
