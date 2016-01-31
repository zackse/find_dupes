package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

var DefaultNumWorkers int = 2
var Excludes = map[string]bool{
	".":         true,
	"..":        true,
	".DS_Store": true,
}

type FileDesc struct {
	path  string
	mtime int64
	size  int64
}

func generateFileDesc(id int, path string) (FileDesc, error) {
	var mTime, size int64
	fileInfo, err := os.Stat(path)
	if err != nil {
		fmt.Printf("worker %d: error os.Stat(%s): %s\n", id, path, err)
		return FileDesc{}, err
	} else {
		mTime = fileInfo.ModTime().Unix()
		size  = fileInfo.Size()
	}
	return FileDesc{path, mTime, size}, nil
}

func getMD5(path string) (string, error) {
	h := md5.New()
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	buf := make([]byte, 1048576)
	for {
		count, err := file.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return "", err
			}
		}
		h.Write(buf[0:count])
	}
	// XXX can call h.checkSum?
	hash := fmt.Sprintf("%x", h.Sum(nil))

	return hash, nil
}

func processFiles(id int, taskQueue <-chan string, doneQueue chan<- map[int64][]FileDesc) {
	filesBySize := make(map[int64][]FileDesc)
	fmt.Printf("worker %d: starting up!\n", id)
	for filename := range taskQueue {
		fileDesc, err := generateFileDesc(id, filename)
		if err != nil {
			fmt.Printf("worker %d: error generating entry for %s: %s\n", id, filename, err)
			continue
		}
		filesBySize[fileDesc.size] = append(filesBySize[fileDesc.size], fileDesc)
	}
	fmt.Printf("worker %d: sending back %d entries\n", id, len(filesBySize))
	doneQueue <- filesBySize
}

func findDupes(dirname string, numWorkers int) map[string][]FileDesc {
	taskQueue := make(chan string, 100)
	doneQueue := make(chan map[int64][]FileDesc)

	for i := 0; i < numWorkers; i++ {
		go processFiles(i, taskQueue, doneQueue)
	}

	//fmt.Printf("going to call filepath.Walk(%s)\n", dirname)
	err := filepath.Walk(dirname, func(path string, info os.FileInfo, err error) error {
		//fmt.Printf("examining path %s\n", path)
		if err != nil {
			fmt.Printf("Error accessing %s: %s\n", path, err)
			return nil
		}
		if _, ok := Excludes[info.Name()]; ok {
			return nil
		}
		if info.Mode().IsRegular() {
			//fmt.Printf("sending path %s to queue\n", path)
			taskQueue <- path
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Error crawling directory tree: %s\n", err)
	}
	close(taskQueue)
	//fmt.Printf("done calling filepath.Walk(%s)\n", dirname)

	filesBySize := make(map[int64][]FileDesc)
	for i := 0; i < numWorkers; i++ {
		workerFilesBySize := <-doneQueue
		for k, v := range workerFilesBySize {
			filesBySize[k] = append(filesBySize[k], v...)
		}
	}

	// walk dupes, get md5s
	filesByHash := make(map[string][]FileDesc)
	for _, fileDescs := range filesBySize {
		if len(fileDescs) < 2 {
			continue
		}
		for i := range fileDescs {
			hash, err := getMD5(fileDescs[i].path)
			if err != nil {
				fmt.Printf("error getMD5(%s): %s\n", fileDescs[i].path, err)
				continue
			} else {
				filesByHash[hash] = append(filesByHash[hash], fileDescs[i])
			}
		}
	}

	return filesByHash
}

func printDupes(dirname string, numWorkers int) {
	filesByHash := findDupes(dirname, numWorkers)
	if len(filesByHash) == 0 {
		fmt.Printf("\nNo dupes found.\n")
		return
	}

	fmt.Printf("\nDuplicates:\n")
	// XXX how to iterate in sorted order
	for hash, fileDescs := range filesByHash {
		if len(fileDescs) < 2 {
			continue
		}
		fmt.Printf("\n%s\n", hash)
		for i := range fileDescs {
			fmt.Printf("\t%s %d\n", fileDescs[i].path, fileDescs[i].mtime)
		}
	}
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
		n, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}
		numWorkers = n
	}
	fmt.Printf("Searching %s with %d workers\n", srcDir, numWorkers)
	printDupes(srcDir, numWorkers)
}
