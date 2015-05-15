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

func checkPatternInFile(filename string, pattern string, ignoreCase bool) bool {
	var re *regexp.Regexp
	if ignoreCase {
		re = regexp.MustCompile(`(?i)` + pattern)
	} else {
		re = regexp.MustCompile(pattern)
	}
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		match := re.MatchString(scanner.Text())
		if match {
			return true
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return false
}

type lineMatch struct {
	filename string
	n        int
	match    [][]string
	end      []string
	line     string
}

func searchAndReplaceInFile(filename string, pattern string, ignoreCase bool) <-chan lineMatch {
	c := make(chan lineMatch)
	var re *regexp.Regexp
	var reEnd *regexp.Regexp
	if ignoreCase {
		re = regexp.MustCompile(`(?i)(.*?)(?P<pattern>` + pattern + `)`)
		reEnd = regexp.MustCompile(`(?i).*` + pattern + `(.*?)$`)
	} else {
		re = regexp.MustCompile(`(.*?)(?P<pattern>` + pattern + `)`)
		reEnd = regexp.MustCompile(`.*` + pattern + `(.*?)$`)
	}
	go func() {
		file, err := os.Open(filename)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		n := 0
		for scanner.Scan() {
			n += 1
			match := re.FindAllStringSubmatch(scanner.Text(), -1)
			remainder := reEnd.FindStringSubmatch(scanner.Text())
			if len(match) != 0 {
				c <- lineMatch{filename: filename, n: n, line: scanner.Text(), match: match, end: remainder}
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
	fmt.Printf("%s%s%s :%s%d%s:%s ", ansi.Magenta, lm.filename, ansi.Blue, ansi.Green, lm.n, ansi.Blue, ansi.Reset)

	// fmt.Printf("%s\n", lm.line)
	// fmt.Printf("%q\n", lm.match)
	for _, m := range lm.match {
		fmt.Printf("%s%s%s%s", m[1], ansi.Red, m[2], ansi.Reset)
	}
	fmt.Printf("%s\n", lm.end[1])
	// fmt.Printf("%q\n", lm.end)

	// for i, n := range lm.match {
	// 	for i, m := range lm.matchNames[i] {
	// 		fmt.Printf("%d. match='%s', name='%s', m='%s'\n", i, n, m, lm.matchNames[i])
	// 	}
	// }
	// fmt.Printf("The names are  : %v\n", lm.matchNames)
	// fmt.Printf("The matches are: %v\n", lm.match)
}

func main() {
	log.Printf("args: %s", os.Args[1:])

	searchBase := os.Args[1]
	pattern := os.Args[2]
	ignoreBinary := true
	ignoreCase := true

	c := getFileList(searchBase, true)

	for filename := range c {
		// fmt.Printf("%s -> %s\n", filename, getMimeType(filename))
		if ignoreBinary == true && !isText(filename) {
			continue
		}
		if checkPatternInFile(filename, pattern, ignoreCase) {
			for d := range searchAndReplaceInFile(filename, pattern, ignoreCase) {
				printLineMatch(d)
			}
		}
	}
}
