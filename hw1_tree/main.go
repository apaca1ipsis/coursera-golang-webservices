package main

import (
	"fmt"
	"io"
	"os"
)

func dirTree(out io.Writer, path string, printFiles bool) error {
	res, e := setFilenames("", path, printFiles)
	if e != nil {
		return e
	}

	for _, s := range res {
		fmt.Fprintln(out, s)
	}

	return nil
}

type FileSorter struct {
	appendToStart bool
	isLastInDir   bool
	isDir         bool
	filePrefix    string
	dirPrefix     string
	filename      string
	printFiles    bool
	path          string
}

func (fS FileSorter) appendToSolution(arr []string, elems []string) []string {
	if fS.appendToStart {
		return append(elems, arr...)
	} else {
		return append(arr, elems...)
	}
}

func (fS *FileSorter) compareNames(prev *string) {
	if fS.filename <= *prev {
		fS.appendToStart = true
	}
	*prev = fS.filename
}

func (fS *FileSorter) checkIsLastInDir(filePos int, dirLen int) {
	fS.isLastInDir = filePos+1 == dirLen
}

func (fS *FileSorter) SetFilePrefix() {
	if fS.isLastInDir {
		fS.filePrefix = "└───"
	} else {
		fS.filePrefix = "├───"
	}
}

func (fS *FileSorter) fileInfo() string {
	return fS.dirPrefix + fS.filePrefix + fS.filename
}

func (fS FileSorter) InnerDirPrefix() string {
	if !fS.isLastInDir {
		return fS.dirPrefix + "│\t"
	} else {
		return fS.dirPrefix + "\t"
	}
}

func (fS FileSorter) InnerDirPath(path string) string {
	return path + string(os.PathSeparator) + fS.filename
}

func (fS *FileSorter) setFile(f os.DirEntry) {
	fS.filename = f.Name()
	fS.isDir = f.IsDir()
}

func (fS *FileSorter) addFileOrDirectory(res *[]string) error {
	if fS.isDir {
		*res = fS.appendToSolution(*res, []string{fS.fileInfo()})

		nwArr, e := setFilenames(
			fS.InnerDirPrefix(),
			fS.InnerDirPath(fS.path),
			fS.printFiles)

		if e != nil {
			return e
		}
		*res = fS.appendToSolution(*res, nwArr)
	} else {
		if fS.printFiles {
			*res = fS.appendToSolution(*res, []string{fS.fileInfo()})
		}

	}
	return nil
}

func setFilenames(prefix string, path string, printFiles bool) ([]string, error) {
	_ = printFiles
	var res []string
	fs, e := os.ReadDir(path)
	if e != nil {
		return nil, e
	}

	var prevWord string
	for i, f := range fs {
		var sorter = FileSorter{dirPrefix: prefix, printFiles: printFiles, path: path}

		sorter.setFile(f)
		sorter.compareNames(&prevWord)
		sorter.checkIsLastInDir(i, len(fs))
		sorter.SetFilePrefix()
		e = sorter.addFileOrDirectory(&res)
		if e != nil {
			return nil, e
		}
	}

	return res, nil
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
