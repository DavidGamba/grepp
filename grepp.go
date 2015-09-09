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

type greppOptions struct {
	ignoreBinary  bool
	caseSensitive bool
	useColor      bool
	useNumber     bool
	filenameOnly  bool
	replace       string
	force         bool
	context       int
	searchBase    string
	// Controls whether or not to show the filename. If the given location is a
	// file then there is no need to show the filename
	showFile          bool
	pattern           string
	filePattern       string
	ignoreFilePattern string
}

func (opt greppOptions) String() string {
	return fmt.Sprintf("ignoreBinary: %v, caseSensitive: %v, useColor %v, useNumber %v, filenameOnly %v, force %v",
		opt.ignoreBinary, opt.caseSensitive, opt.useColor, opt.useNumber, opt.filenameOnly, opt.force)
}

func grepp(ow io.Writer, opt greppOptions) {
	c := getFileList(opt.searchBase, true)

	for filename := range c {
		// fmt.Printf("%s -> %s\n", filename, getMimeType(filename))
		if opt.ignoreBinary == true && !isText(filename) {
			continue
		}
		if opt.filenameOnly {
			if checkPatternInFile(filename, opt.pattern, !opt.caseSensitive) {
				l.Info.Printf("%s%s\n", color(ansi.Magenta, filename, opt.useColor), colorReset(opt.useColor))
			}
		} else {
			if checkPatternInFile(filename, opt.pattern, !opt.caseSensitive) {
				var tmpFile *os.File
				var err error
				if opt.force {
					tmpFile, err = ioutil.TempFile("", filepath.Base(filename)+"-")
					defer tmpFile.Close()
					if err != nil {
						l.Error.Println("cannot open ", tmpFile)
						l.Error.Fatal(err)
					}
					l.Debug.Printf("tmpFile: %v", tmpFile.Name())
				}
				for d := range searchAndReplaceInFile(filename, opt.pattern, !opt.caseSensitive) {
					if len(d.match) == 0 {
						if opt.context > 0 {
							printLineContext(ow, d, opt.useColor, opt.useNumber, opt.showFile)
						}
					} else {
						printLineMatch(ow, d, opt.useColor, opt.useNumber, opt.replace, opt.showFile)
					}
					if opt.force {
						if len(d.match) == 0 {
							tmpFile.WriteString(d.line + "\n")
						} else {
							writeLineMatch(tmpFile, d, opt.replace)
						}
					}
				}
				if opt.force {
					tmpFile.Close()
					err = copyFileContents(tmpFile.Name(), filename)
					if err != nil {
						l.Warning.Printf("Couldn't update file: %s. '%s'\n", filename, err)
					}
				}
			}
		}
	}
}

func runInPager(fn func(io.Writer, greppOptions), opt greppOptions) {
	pager := strings.Split(os.Getenv("PAGER"), " ")
	var cmd *exec.Cmd
	// Make sure to use -R to show colors when using less
	if pager[0] == "less" {
		pager[0] = "-R"
		cmd = exec.Command("less", pager...)
	} else {
		cmd = exec.Command(pager[0], pager[1:]...)
	}
	var pr *io.PipeReader
	var pw *io.PipeWriter
	// create a pipe (blocking)
	pr, pw = io.Pipe()
	cmd.Stdin = pr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cPager := make(chan struct{})
	// Create a blocking chan, Run the pager and unblock once it is finished
	go func() {
		cmd.Run()
		close(cPager)
		os.Exit(0)
	}()

	fn(pw, opt)

	// Close pipe
	pw.Close()

	// Wait for the pager to be finished
	<-cPager
}

func main() {
	l.LogInit(ioutil.Discard, ioutil.Discard, os.Stdout, os.Stderr, os.Stderr)
	l.Debug.Printf("args: %s", os.Args[1:])

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

	opt := greppOptions{}
	opt.ignoreBinary = options["I"].(bool)
	opt.caseSensitive = options["c"].(bool)
	opt.useColor = options["color"].(bool)
	opt.useNumber = options["n"].(bool)
	opt.filenameOnly = options["l"].(bool)
	opt.replace = options["r"].(string)
	opt.force = options["f"].(bool)
	opt.context = options["C"].(int)

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
	if len(remaining) < 2 {
		opt.searchBase = "."
	} else {
		opt.searchBase = remaining[1]
	}
	searchBaseInfo, err := os.Stat(opt.searchBase)
	if err != nil {
		l.Error.Println("cannot stat", opt.searchBase)
		l.Error.Fatal(err)
	}
	if searchBaseInfo.IsDir() {
		opt.showFile = true
	} else {
		opt.showFile = false
	}

	opt.pattern = remaining[0]

	l.Debug.Printf("pattern: %s, searchBase: %s, replace: %s", opt.pattern, opt.searchBase, opt.replace)
	l.Debug.Printf(fmt.Sprintln(opt))

	// Check if stdout is pipe p or device D
	statStdout, _ := os.Stdout.Stat()
	l.Debug.Printf("stats Stdout: %s", statStdout.Mode())
	if (statStdout.Mode() & os.ModeNamedPipe) == 0 {
		runInPager(grepp, opt)
	} else {
		opt.useColor = false
		grepp(os.Stdout, opt)
	}
}
