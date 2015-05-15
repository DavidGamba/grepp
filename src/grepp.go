package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func getMimeType(filename string) string {
	file, err := os.Open(filename)
	if err != nil {
		println("cannot open", filename)
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Scan()
	b := scanner.Bytes()
	return http.DetectContentType(b)
}

func isText(filename string) bool {
	s := getMimeType(filename)
	return strings.HasPrefix(s, "text/")
}

func getFileList(filename string, c chan string, ignoreDirs bool) {
	log.Printf("getFileList: %s", filename)
	fInfo, err := os.Stat(filename)
	if err != nil {
		println("cannot stat", filename)
		log.Fatal(err)
	}
	if fInfo.IsDir() {
		if ignoreDirs == false {
			c <- filename
		}
		fileSearch := filename + string(filepath.Separator) + "*"
		log.Printf("file search: %s", fileSearch)
		fileMatches, err := filepath.Glob(fileSearch)
		if err != nil {
			println("error: ", err)
			log.Fatal(err)
		}
		log.Printf("fileMatches: %s", fileMatches)
		fileChannels := make([]chan string, 0)
		for _, file := range fileMatches {
			if filepath.Base(filename) == filepath.Base(file) {
				log.Printf("skipping: %s", filename)
				continue
			}
			log.Printf("go: %s", file)
			fileChannels = append(fileChannels, make(chan string))
			go getFileList(file, fileChannels[len(fileChannels)-1], ignoreDirs)
			c <- <-fileChannels[len(fileChannels)-1]
		}
	} else {
		c <- filename
	}
	// close(c)
}

func scanFile(filename string, pattern string, c chan string) {
	re := regexp.MustCompile(pattern)
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		match := re.FindStringSubmatch(scanner.Text())
		if len(match) != 0 {
			// fmt.Printf("1. %s\n", match[1])
			c <- filename + " : " + scanner.Text()
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	close(c)
}

func main() {
	log.Printf("args: %s", os.Args[1:])

	searchBase := os.Args[1]
	// pattern := os.Args[2]
	ignoreBinary := true

	c := make(chan string)
	go getFileList(searchBase, c, true)

	// sliceFileRead := make([]chan string, 0)
	// log.Printf("slice: %s", sliceFileRead)
	for filename := range c {
		fmt.Printf("%s -> %s\n", filename, getMimeType(filename))
		if ignoreBinary == true && !isText(filename) {
			continue
		}
		// sliceFileRead = append(sliceFileRead, make(chan string))
		// log.Printf("slice 2: %v, %v", sliceFileRead, len(sliceFileRead))
		// go scanFile(filename, pattern, sliceFileRead[len(sliceFileRead)-1])
	}
	// log.Printf("slice 3: %v", sliceFileRead)
	// for d := range sliceFileRead {
	// 	for r := range sliceFileRead[d] {
	// 		fmt.Printf("%s\n", r)
	// 	}
	// }
}
