package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
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
	hash  string
}

func generateFileDesc(id int, path string) (FileDesc, error) {
	h := md5.New()
	file, err := os.Open(path)
	if err != nil {
		return FileDesc{}, err
	}
	defer file.Close()

	buf := make([]byte, 4096)
	for {
		count, err := file.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return FileDesc{}, err
			}
		}
		h.Write(buf[0:count])
	}
	// XXX can call h.checkSum?
	hash := fmt.Sprintf("%x", h.Sum(nil))

	var mTime int64
	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Printf("worker %d: error os.Stat(%s): %s\n", id, path, err)
		mTime = 0
	} else {
		mTime = fileInfo.ModTime().Unix()
	}
	return FileDesc{path, mTime, hash}, nil
}

func processFiles(id int, taskQueue <-chan string, doneChan chan<- bool, filesByHash map[string][]FileDesc, mutex *sync.Mutex) {
	fmt.Printf("worker %d: starting up!\n", id)
	for filename := range taskQueue {
		fileDesc, err := generateFileDesc(id, filename)
		if err != nil {
			fmt.Printf("worker %d: error generating entry for %s: %s\n", id, filename, err)
			continue
		}
		mutex.Lock()
		filesByHash[fileDesc.hash] = append(filesByHash[fileDesc.hash], fileDesc)
		mutex.Unlock()
	}
	doneChan <- true
}

func findDupes(dirname string, numWorkers int) map[string][]FileDesc {
	taskQueue := make(chan string, 100)
	doneChan := make(chan bool)
	mutex := &sync.Mutex{}
	filesByHash := make(map[string][]FileDesc)

	for i := 0; i < numWorkers; i++ {
		go processFiles(i, taskQueue, doneChan, filesByHash, mutex)
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
	for i := 0; i < numWorkers; i++ {
		<-doneChan
	}

	return filesByHash
}

func printDupes(dirname string, numWorkers int) {
	filesByHash := findDupes(dirname, numWorkers)
	if len(filesByHash) == 0 {
		fmt.Printf("\nNo dupes found.\n")
		return
	}

	fmt.Printf("\nDupes found:\n")
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
