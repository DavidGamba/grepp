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

	reader := bufio.NewReaderSize(file, 4*1024)

	for {
		line, isPrefix, err := reader.ReadLine()
		if isPrefix {
			fmt.Println(errors.New(filename + ": buffer size to small"))
			break
		}
		match := re.MatchString(string(line))
		if match {
			return true
		}
		if err != nil {
			if err != io.EOF {
				fmt.Printf("ERROR: %v\n", err)
			}
			break
		}
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

func color(color string, line string, useColor bool) string {
	if useColor {
		return fmt.Sprintf("%s%s", color, line)
	} else {
		return fmt.Sprintf("%s", line)
	}
}

func colorReset(useColor bool) string {
	if useColor {
		return fmt.Sprintf("%s", ansi.Reset)
	} else {
		return ""
	}
}

// Each section is in charge of starting with the color or reset.
func printLineMatch(lm lineMatch, useColor bool, useNumber bool) {
	stringLine := func() string {
		if useColor {
			result := ansi.Reset
			for _, m := range lm.match {
				result += fmt.Sprintf("%s%s%s%s", m[1], ansi.Red, m[2], ansi.Reset)
			}
			result += lm.end[1]
			return result
		} else {
			return lm.line
		}
	}

	result := color(ansi.Magenta, lm.filename, useColor) + " " + color(ansi.Blue, ":", useColor)
	if useNumber {
		result += color(ansi.Green, strconv.Itoa(lm.n), useColor) + color(ansi.Blue, ":", useColor)
	}
	result += colorReset(useColor) + " " + stringLine()
	fmt.Println(result)
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
