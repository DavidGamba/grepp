/*
grepp - An improved version of the most common combinations of grep, find and sed in a single script.
*/
package main

import (
	"bufio"
	"errors"
	"fmt"
	gopt "github.com/davidgamba/grepp/getoptions"
	l "github.com/davidgamba/grepp/logging"
	"github.com/mgutz/ansi"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func getMimeType(filename string) string {
	file, err := os.Open(filename)
	if err != nil {
		l.Warning.Println("cannot open", filename)
		l.Error.Fatal(err)
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
	l.Trace.Printf("getFileList: %s", filename)
	c := make(chan string)
	go func() {
		fInfo, err := os.Stat(filename)
		if err != nil {
			l.Warning.Println("cannot stat", filename)
			l.Error.Fatal(err)
		}
		if fInfo.IsDir() {
			if ignoreDirs == false {
				c <- filename
			}
			fileSearch := filename + string(filepath.Separator) + "*"
			l.Trace.Printf("file search: %s", fileSearch)
			fileMatches, err := filepath.Glob(fileSearch)
			if err != nil {
				l.Error.Fatal(err)
			}
			l.Trace.Printf("fileMatches: %s", fileMatches)
			for _, file := range fileMatches {
				if filepath.Base(filename) == filepath.Base(file) {
					l.Debug.Printf("skipping: %s", filename)
					continue
				}
				l.Trace.Printf("go: %s", file)
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
		l.Error.Fatal(err)
	}
	defer file.Close()

	reader := bufio.NewReaderSize(file, 4*1024)

	for {
		line, isPrefix, err := reader.ReadLine()
		if isPrefix {
			l.Warning.Println(errors.New(filename + ": buffer size to small"))
			break
		}
		if err != nil {
			if err != io.EOF {
				l.Error.Println(err)
			}
			break
		}
		match := re.MatchString(string(line))
		if match {
			return true
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
			l.Error.Fatal(err)
		}
		defer file.Close()

		reader := bufio.NewReaderSize(file, 4*1024)
		// line number
		n := 0
		for {
			n += 1
			line, isPrefix, err := reader.ReadLine()
			if isPrefix {
				l.Warning.Println(errors.New(filename + ": buffer size to small"))
				break
			}
			// stop reading file
			if err != nil {
				if err != io.EOF {
					l.Error.Println(err)
				}
				break
			}
			match := re.FindAllStringSubmatch(string(line), -1)
			remainder := reEnd.FindStringSubmatch(string(line))
			c <- lineMatch{filename: filename, n: n, line: string(line), match: match, end: remainder}
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

func writeLineMatch(file *os.File, lm lineMatch, replace string) {
	for _, m := range lm.match {
		file.WriteString(m[1] + replace)
	}
	file.WriteString(lm.end[1] + "\n")
}

// Each section is in charge of starting with the color or reset.
func printLineMatch(w io.Writer, lm lineMatch, useColor, useNumber bool, replace string, showFile bool) {
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

	result := ""
	if showFile {
		result += color(ansi.Magenta, lm.filename, useColor) + " " + color(ansi.Blue, ":", useColor)
	}
	if useNumber {
		result += color(ansi.Green, strconv.Itoa(lm.n), useColor) + color(ansi.Blue, ":", useColor)
	}
	result += colorReset(useColor) + " " + stringLine()
	fmt.Fprintln(w, result)
}

// Each section is in charge of starting with the color or reset.
func printLineContext(w io.Writer, lm lineMatch, useColor, useNumber bool, showFile bool) {
	result := ""
	if showFile {
		result += color(ansi.Magenta, lm.filename, useColor) + " " + color(ansi.Blue, "-", useColor)
	}
	if useNumber {
		result += color(ansi.Green, strconv.Itoa(lm.n), useColor) + color(ansi.Blue, "-", useColor)
	}
	result += colorReset(useColor) + " " + lm.line
	fmt.Fprintln(w, result)
}

// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

func main() {
	l.LogInit(ioutil.Discard, ioutil.Discard, os.Stdout, os.Stderr, os.Stderr)
	l.Debug.Printf("args: %s", os.Args[1:])

	// Controls whether or not to show the filename. If the given location is a
	// file then there is no need to show the filename
	var showFile bool
	var replace string
	// var filePattern string
	// var ignoreFilePattern string

	options, remaining := gopt.GetOptLong(os.Args[1:], "normal",
		gopt.OptDef{
			"I":       {"", true},  // ignoreBinary
			"c":       {"", false}, // caseSensitive
			"color":   {"", true},  // useColor
			"n":       {"", true},  // useNumber
			"l":       {"", false}, // filenameOnly
			"r":       {"=s", ""},  // replace
			"f":       {"", false}, // force
			"C":       {"=i", 0},   // context
			"fp":      {"", false}, // fullPath - Used to show the file full path instead of the relative to the current dir.
			"name":    {"=s", ""},  // filePattern - Use to further filter the search to files matching that pattern.
			"ignore":  {"=s", ""},  // ignoreFilePattern - Use to further filter the search to files not matching that pattern.
			"spacing": {"", false}, // keepSpacing - Do not remove initial spacing.
			"debug":   {"", false}, // debug logging
			"trace":   {"", false}, // trace logging
		},
	)

	ignoreBinary := options["I"].(bool)
	caseSensitive := options["c"].(bool)
	useColor := options["color"].(bool)
	useNumber := options["n"].(bool)
	filenameOnly := options["l"].(bool)
	replace = options["r"].(string)
	force := options["f"].(bool)
	context := options["C"].(int)

	debug := options["debug"].(bool)
	trace := options["trace"].(bool)
	if debug {
		l.LogInit(ioutil.Discard, os.Stderr, os.Stdout, os.Stderr, os.Stderr)
	}
	if trace {
		l.LogInit(os.Stderr, os.Stderr, os.Stdout, os.Stderr, os.Stderr)
	}

	if len(remaining) < 1 {
		l.Error.Fatal("Missing pattern!")
	}
	var searchBase string
	if len(remaining) < 2 {
		searchBase = "."
	} else {
		searchBase = remaining[1]
	}
	searchBaseInfo, err := os.Stat(searchBase)
	if err != nil {
		l.Error.Println("cannot stat", searchBase)
		l.Error.Fatal(err)
	}
	if searchBaseInfo.IsDir() {
		showFile = true
	} else {
		showFile = false
	}

	pattern := remaining[0]

	l.Debug.Printf("pattern: %s, searchBase: %s, replace: %s", pattern, searchBase, replace)
	l.Debug.Printf("ignoreBinary: %v, caseSensitive: %v, useColor %v, useNumber %v, filenameOnly %v, force %v",
		ignoreBinary, caseSensitive, useColor, useNumber, filenameOnly, force)

	pager := strings.Split(os.Getenv("PAGER"), " ")
	var cmd *exec.Cmd
	// Make sure to use -R to show colors when using less
	if pager[0] == "less" {
		pager[0] = "-R"
		cmd = exec.Command("less", pager...)
	} else {
		cmd = exec.Command(pager[0], pager[1:]...)
	}
	// ow output writer
	var ow io.Writer
	var pr *io.PipeReader
	var pw *io.PipeWriter
	// create a pipe (blocking)
	pr, pw = io.Pipe()
	cmd.Stdin = pr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cPager := make(chan struct{})
	// Check if stdout if pipe p or device D
	statStdout, _ := os.Stdout.Stat()
	if (statStdout.Mode() & os.ModeNamedPipe) == 0 {
		fmt.Println(statStdout)
		ow = pw
		// Create a blocking chan, Run the pager and unblock once it is finished
		go func() {
			cmd.Run()
			close(cPager)
			os.Exit(0)
		}()
	} else {
		close(cPager)
		ow = os.Stdout
	}

	c := getFileList(searchBase, true)

	for filename := range c {
		// fmt.Printf("%s -> %s\n", filename, getMimeType(filename))
		if ignoreBinary == true && !isText(filename) {
			continue
		}
		if filenameOnly {
			if checkPatternInFile(filename, pattern, !caseSensitive) {
				l.Info.Printf("%s%s\n", color(ansi.Magenta, filename, useColor), colorReset(useColor))
			}
		} else {
			if checkPatternInFile(filename, pattern, !caseSensitive) {
				var tmpFile *os.File
				var err error
				if force {
					tmpFile, err = ioutil.TempFile("", filepath.Base(filename)+"-")
					defer tmpFile.Close()
					if err != nil {
						l.Error.Println("cannot open ", tmpFile)
						l.Error.Fatal(err)
					}
					l.Debug.Printf("tmpFile: %v", tmpFile.Name())
				}
				for d := range searchAndReplaceInFile(filename, pattern, !caseSensitive) {
					if len(d.match) == 0 {
						if context > 0 {
							printLineContext(ow, d, useColor, useNumber, showFile)
						}
					} else {
						printLineMatch(ow, d, useColor, useNumber, replace, showFile)
					}
					if force {
						if len(d.match) == 0 {
							tmpFile.WriteString(d.line + "\n")
						} else {
							writeLineMatch(tmpFile, d, replace)
						}
					}
				}
				if force {
					tmpFile.Close()
					err = copyFileContents(tmpFile.Name(), filename)
					if err != nil {
						l.Warning.Printf("Couldn't update file: %s. '%s'\n", filename, err)
					}
				}
			}
		}
	}

	// Close pipe
	pw.Close()

	// Wait for the pager to be finished
	<-cPager
}
