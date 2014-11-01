package main

import (
	"crypto/md5"
	"filepath"
	"fmt"
	"io"
	"os"
	"strconv"
)

var DefaultNumWorkers int = 2
var Excludes = map[string]boolean{
	".":         true,
	"..":        true,
	".DS_Store": true,
}

type FileDesc struct {
	path  string
	mtime int
	hash  string
}

func digestForFile(id int, path string) FileDesc {
	h := md5.New()
	file, err := os.Open(path)
	if err != nil {
		fmt.Println("worker %d: error os.Open(%s): %s\n", id, filename, err)
		return
	}
	buf := make([]byte, 1048576)
	for {
		count, err := file.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				fmt.Println("worker %d: error Read(%s): %s\n", id, filename, err)
				return
			}
		}
		h.Write(buf[0:count])
	}
	// XXX can call h.checkSum?
	hash := fmt.Sprintf("%x", h.Sum(nil))

	var mTime int
	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Println("worker %d: error os.Stat(%s): %s\n", id, filename, err)
		mTime = 0
	} else {
		mTime = fileInfo.ModTime().Second()
	}
	return FileDesc{filename, hash, mTime}
}

func processFiles(id int, taskQueue <-chan string, doneQueue chan<- map[string][]string) {
	filesByHash := make(map[string][]FileDesc)
	fmt.Println("worker %d: starting up!\n", id)
	for filename := range taskQueue {
		filesByHash[h] = append(filesByHash[h], generateFileEntry(id, filename))
	}
}

func findDupes(dirname string, numWorkers int) map[string][]string {
	taskQueue := make(chan string, 100)
	doneQueue := make(chan map[string][]FileDesc)

	for i := 0; i < numWorkers; i++ {
		go processFiles(i, taskQueue, doneQueue)
	}

	err := filepath.Walk(dirname, func(string fileOrDir) {
		if _, ok := Excludes[fileOrDir]; ok {
			continue
		}
		isDir, err := IsDirectory(fileOrDir)
		if err != nil {
			fmt.Printf("Error accessing %s: %s\n", fileOrDir, err)
			return
		}
		if !isDir {
			taskQueue <- filepath.Join(root, f)
		}
	})
	if err != nil {
		fmt.Printf("Error crawling directory tree: %s\n", err)
	}
	close(taskQueue)

	filesByHash := make(map[string][]FileDesc)
	for i := 0; i < numWorkers; i++ {
		workerFilesByHash := <-doneQueue
		for k, v := range workerFilesByHash {
			_, ok := filesByHash[k]
			if !ok {
				filesByHash[k] = make([]string, 5)
			}
			filesByHash[k] = append(filesByHash[k], v)
		}
	}
	return filesByHash
}

func printDupes(dirname string, numWorkers int) {
	filesByHash := findDupes(dirname, numWorkers)
}

// adapted from http://stackoverflow.com/a/25567952
func IsDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), err
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: find_dupes <PATH> [ <NUM_WORKERS> ]\n")
		os.Exit(1)
	}

	srcDir := os.Args[1]
	if isDir, err := IsDirectory(srcDir); err != nil || !isDir {
		fmt.Printf("%s is not a directory\n", srcDir)
		os.Exit(1)
	}

	numWorkers := DefaultNumWorkers
	if len(os.Args) > 2 {
		numWorkers, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	fmt.Printf("Searching %s with %d workers\n", srcDir, numWorkers)

}
