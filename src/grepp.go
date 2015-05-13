package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	// log.Printf("filename: %s", filename)
	fInfo, err := os.Stat(filename)
	if err != nil {
		println("cannot stat", filename)
		log.Fatal(err)
	}
	if fInfo.IsDir() {
		if ignoreDirs == false {
			c <- filename
		}
		fileMatches, err := filepath.Glob(filename + string(filepath.Separator) + "*")
		if err != nil {
			println("error: ", err)
			log.Fatal(err)
		}
		for _, file := range fileMatches {
			if filepath.Base(filename) == filepath.Base(file) {
				continue
			}
			// println("result: " + file)
			d := make(chan string)
			go getFileList(file, d, ignoreDirs)
			c <- <-d
		}
	} else {
		c <- filename
	}
	close(c)
}

func main() {
	log.Printf("args: %s", os.Args[1:])
	c := make(chan string)
	for _, v := range os.Args[1:] {
		go getFileList(v, c, true)
	}
	for filename := range c {
		fmt.Printf("%s -> %s\n", filename, getMimeType(filename))
	}
}
