/*
grepp - An improved version of the most common combinations of grep, find and sed in a single script.
*/
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"github.com/mgutz/ansi"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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
	re, _ := getRegex(pattern, ignoreCase)
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

func getRegex(pattern string, ignoreCase bool) (re, reEnd *regexp.Regexp) {
	if ignoreCase {
		re = regexp.MustCompile(`(?i)(.*?)(?P<pattern>` + pattern + `)`)
		reEnd = regexp.MustCompile(`(?i).*` + pattern + `(.*?)$`)
	} else {
		re = regexp.MustCompile(`(.*?)(?P<pattern>` + pattern + `)`)
		reEnd = regexp.MustCompile(`.*` + pattern + `(.*?)$`)
	}
	return
}

func searchAndReplaceInFile(filename, pattern string, ignoreCase bool) <-chan lineMatch {
	c := make(chan lineMatch)
	re, reEnd := getRegex(pattern, ignoreCase)
	go func() {
		file, err := os.Open(filename)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		reader := bufio.NewReaderSize(file, 4*1024)
		// line number
		n := 0
		for {
			n += 1
			line, isPrefix, err := reader.ReadLine()
			if isPrefix {
				fmt.Println(errors.New(filename + ": buffer size to small"))
				break
			}
			match := re.FindAllStringSubmatch(string(line), -1)
			remainder := reEnd.FindStringSubmatch(string(line))
			c <- lineMatch{filename: filename, n: n, line: string(line), match: match, end: remainder}
			// stop reading file
			if err != nil {
				if err != io.EOF {
					fmt.Printf("ERROR: %v\n", err)
				}
				break
			}
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

//TODO: Don't drop the control char but scape it and show it like less.

// http://rosettacode.org/wiki/Strip_control_codes_and_extended_characters_from_a_string#Go
// two UTF-8 functions identical except for operator comparing c to 127
func stripCtlFromUTF8(str string) string {
	return strings.Map(func(r rune) rune {
		if r >= 32 && r != 127 {
			return r
		}
		return -1
	}, str)
}

// Each section is in charge of starting with the color or reset.
func printLineMatch(lm lineMatch, useColor, useNumber bool, replace string) {
	stringLine := func() string {
		if useColor {
			result := ansi.Reset
			for _, m := range lm.match {
				result += fmt.Sprintf("%s%s%s%s%s%s",
					stripCtlFromUTF8(m[1]),
					ansi.Red,
					stripCtlFromUTF8(m[2]),
					ansi.Green,
					stripCtlFromUTF8(replace),
					ansi.Reset)
			}
			result += stripCtlFromUTF8(lm.end[1])
			return result
		} else {
			return stripCtlFromUTF8(lm.line)
		}
	}

	result := color(ansi.Magenta, lm.filename, useColor) + " " + color(ansi.Blue, ":", useColor)
	if useNumber {
		result += color(ansi.Green, strconv.Itoa(lm.n), useColor) + color(ansi.Blue, ":", useColor)
	}
	result += colorReset(useColor) + " " + stringLine()
	fmt.Println(result)
}

// Each section is in charge of starting with the color or reset.
func printLineContext(lm lineMatch, useColor, useNumber bool) {
	result := color(ansi.Magenta, lm.filename, useColor) + " " + color(ansi.Blue, "-", useColor)
	if useNumber {
		result += color(ansi.Green, strconv.Itoa(lm.n), useColor) + color(ansi.Blue, "-", useColor)
	}
	result += colorReset(useColor) + " " + lm.line
	fmt.Println(result)
}

func main() {
	log.Printf("args: %s", os.Args[1:])

	var ignoreBinary bool
	var caseSensitive bool
	var useColor bool
	var filenameOnly bool
	var useNumber bool
	var replace string
	var force bool
	var context int
	// ignoreBinary := true
	// ignoreCase := true
	// useColor := true
	// filenameOnly := false
	// useNumber := true

	flag.BoolVar(&ignoreBinary, "I", true, "ignore-binary")
	flag.BoolVar(&caseSensitive, "c", false, "case-sensitive")
	flag.BoolVar(&useColor, "color", true, "color")
	flag.BoolVar(&useNumber, "n", true, "use-number")
	flag.BoolVar(&filenameOnly, "l", false, "filename-only")
	flag.StringVar(&replace, "r", "", "replace")
	flag.BoolVar(&force, "f", false, "force")
	flag.IntVar(&context, "C", 0, "context")
	flag.Parse()
	log.Printf("flag args: %s, n %v", flag.Args(), flag.NArg())

	if flag.NArg() < 1 {
		log.Printf("Missing pattern")
		os.Exit(1)
	}
	var searchBase string
	if flag.NArg() < 2 {
		searchBase = "."
	} else {
		searchBase = flag.Args()[1]
	}

	pattern := flag.Args()[0]

	log.Printf("pattern: %s, searchBase: %s, replace: %s", pattern, searchBase, replace)
	log.Printf("ignoreBinary: %v, caseSensitive: %v, useColor %v, useNumber %v, filenameOnly %v, force %v",
		ignoreBinary, caseSensitive, useColor, useNumber, filenameOnly, force)

	c := getFileList(searchBase, true)

	for filename := range c {
		// fmt.Printf("%s -> %s\n", filename, getMimeType(filename))
		if ignoreBinary == true && !isText(filename) {
			continue
		}
		if filenameOnly {
			if checkPatternInFile(filename, pattern, !caseSensitive) {
				fmt.Printf("%s%s\n", color(ansi.Magenta, filename, useColor), colorReset(useColor))
			}
		} else {
			if checkPatternInFile(filename, pattern, !caseSensitive) {
				if force {
					tmpFile, err := ioutil.TempFile("", filepath.Base(filename)+"-")
					// defer os.Remove(tmpFile.Name())
					if err != nil {
						println("cannot open ", tmpFile)
						log.Fatal(err)
					}
					log.Printf("tmpFile: %v", tmpFile.Name())
				}
				for d := range searchAndReplaceInFile(filename, pattern, !caseSensitive) {
					if len(d.match) == 0 {
						if context > 0 {
							printLineContext(d, useColor, useNumber)
						}
					} else {
						printLineMatch(d, useColor, useNumber, replace)
					}
					if force {
						// TODO: This is how I would get non-matching lines to print to the file.
						// A new function needs to be created to print matched sections to file with the replacement and without color.
						if len(d.match) == 0 {
							printLineContext(d, useColor, useNumber)
						} else {
							printLineMatch(d, useColor, useNumber, replace)
						}
					}
				}
			}
		}
	}
}
