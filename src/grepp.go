package main

import (
	"bufio"
	"fmt"
	"github.com/mgutz/ansi"
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

func getFileList(filename string, ignoreDirs bool) <-chan string {
	// log.Printf("getFileList: %s", filename)
	c := make(chan string)
	go func() {
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
			// log.Printf("file search: %s", fileSearch)
			fileMatches, err := filepath.Glob(fileSearch)
			if err != nil {
				println("error: ", err)
				log.Fatal(err)
			}
			// log.Printf("fileMatches: %s", fileMatches)
			for _, file := range fileMatches {
				if filepath.Base(filename) == filepath.Base(file) {
					log.Printf("skipping: %s", filename)
					continue
				}
				// log.Printf("go: %s", file)
				d := getFileList(file, ignoreDirs)
				for dirFile := range d {
					c <- dirFile
				}
			}
		} else {
			c <- filename
		}
		close(c)
	}()
	return c
}

type lineMatch struct {
	filename string
	n        int
	match    string
}

func scanFile(filename string, pattern string) <-chan lineMatch {
	c := make(chan lineMatch)
	go func() {
		re := regexp.MustCompile(pattern)
		file, err := os.Open(filename)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		n := 0
		for scanner.Scan() {
			n += 1
			match := re.FindStringSubmatch(scanner.Text())
			if len(match) != 0 {
				// fmt.Printf("1. %s\n", match[1])
				c <- lineMatch{filename: filename, n: n, match: scanner.Text()}
			}
		}

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
		close(c)
	}()
	return c
}

func printLineMatch(lm lineMatch) {
	fmt.Printf("%s%s%s :%s%d%s:%s %s\n", ansi.Magenta, lm.filename, ansi.Blue, ansi.Green, lm.n, ansi.Blue, ansi.Reset, lm.match)
}

func main() {
	log.Printf("args: %s", os.Args[1:])

	searchBase := os.Args[1]
	pattern := os.Args[2]
	ignoreBinary := true

	c := getFileList(searchBase, true)

	for filename := range c {
		// fmt.Printf("%s -> %s\n", filename, getMimeType(filename))
		if ignoreBinary == true && !isText(filename) {
			continue
		}
		for d := range scanFile(filename, pattern) {
			printLineMatch(d)
		}
	}
}
