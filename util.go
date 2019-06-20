package gotfp

import (
	"os"
	"sort"
)

func GetFileInfo(path string) FileInfo {
	info, err := os.Lstat(path)
	var category FileCategory
	var childrenNames []string
	if err != nil || info == nil {
		category = ErrorFile
	} else if info.Mode()&os.ModeSymlink != 0 {
		category = Symlink
	} else if info.IsDir() {
		// Get the name of files under this directory.
		childrenNames, err = readDirNames(path)
		if err == nil {
			category = Directory
		} else {
			category = ErrorFile
		}
	} else if info.Mode().IsRegular() {
		category = RegularFile
	} else {
		category = OtherFile
	}
	return FileInfo{
		Path:  path,
		Cat:   category,
		Info:  info,
		Chldn: childrenNames,
		Err:   err,
	}
}

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
