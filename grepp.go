/*
grepp - An improved version of the most common combinations of grep, find and sed in a single script.
*/
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/davidgamba/go-getoptions"
	l "github.com/davidgamba/grepp/logging"
	"github.com/davidgamba/grepp/runInPager"
	"github.com/davidgamba/grepp/semver"
	"github.com/mgutz/ansi"
)

// Buffer Size used to read files when searching through them.
// Default value should cover most cases.
var bufferSize = 4 * 1024

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

var errorBufferSizeTooSmall = fmt.Errorf("buffer size too small")

func checkPatternInFile(filename string, pattern string, ignoreCase bool) (bool, error) {
	re, _ := getRegex(pattern, ignoreCase)
	file, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer file.Close()

	reader := bufio.NewReaderSize(file, bufferSize)

	var errRet error
	for {
		line, isPrefix, err := reader.ReadLine()
		if isPrefix {
			errRet = errorBufferSizeTooSmall
			break
		}
		if err != nil {
			if err != io.EOF {
				errRet = err
			}
			break
		}
		match := re.MatchString(string(line))
		if match {
			return true, errRet
		}
	}
	return false, errRet
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

func searchInFile(filename, pattern string, ignoreCase bool) <-chan lineMatch {
	c := make(chan lineMatch)
	re, reEnd := getRegex(pattern, ignoreCase)
	go func() {
		file, err := os.Open(filename)
		if err != nil {
			l.Error.Fatal(err)
		}
		defer file.Close()

		reader := bufio.NewReaderSize(file, bufferSize)
		// line number
		n := 0
		for {
			n++
			line, isPrefix, err := reader.ReadLine()
			if isPrefix {
				l.Warning.Println(errors.New(filename + ": buffer size too small"))
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
	}
	return fmt.Sprintf("%s", line)
}

func colorReset(useColor bool) string {
	if useColor {
		return fmt.Sprintf("%s", ansi.Reset)
	}
	return ""
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

func (g grepp) writeLineMatch(file *os.File, lm lineMatch) {
	for _, m := range lm.match {
		file.WriteString(m[1] + g.replace)
	}
	file.WriteString(lm.end[1] + "\n")
}

// Each section is in charge of starting with the color or reset.
func (g grepp) printLineMatch(lm lineMatch) {
	stringLine := func() string {
		if g.useColor {
			result := ansi.Reset
			for _, m := range lm.match {
				result += fmt.Sprintf("%s%s%s%s%s%s",
					stripCtlFromUTF8(m[1]),
					ansi.Red,
					stripCtlFromUTF8(m[2]),
					ansi.Green,
					stripCtlFromUTF8(g.replace),
					ansi.Reset)
			}
			result += stripCtlFromUTF8(lm.end[1])
			return result
		}
		return stripCtlFromUTF8(lm.line)
	}

	result := ""
	if g.showFile {
		result += color(ansi.Magenta, lm.filename, g.useColor) + " " + color(ansi.Blue, ":", g.useColor)
	}
	if g.useNumber {
		result += color(ansi.Green, strconv.Itoa(lm.n), g.useColor) + color(ansi.Blue, ":", g.useColor)
	}
	result += colorReset(g.useColor) + " " + stringLine()
	fmt.Fprintln(g.Stdout, result)
}

// Each section is in charge of starting with the color or reset.
func (g grepp) printMinorWarning(line string) {
	result := color(ansi.LightBlack, line, g.useColor)
	fmt.Fprintln(g.Stderr, result)
}

// Each section is in charge of starting with the color or reset.
func (g grepp) printLineContext(lm lineMatch) {
	result := ""
	if g.showFile {
		result += color(ansi.Magenta, lm.filename, g.useColor) + " " + color(ansi.Blue, "-", g.useColor)
	}
	if g.useNumber {
		result += color(ansi.Green, strconv.Itoa(lm.n), g.useColor) + color(ansi.Blue, "-", g.useColor)
	}
	result += colorReset(g.useColor) + " " + lm.line
	fmt.Fprintln(g.Stdout, result)
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

type grepp struct {
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
	showFile             bool
	showBufferSizeErrors bool
	bufferSizeErrorsC    int
	pattern              string
	filePattern          string
	ignoreFilePattern    string
	Stdout               io.Writer
	Stderr               io.Writer
}

func (g grepp) String() string {
	return fmt.Sprintf("ignoreBinary: %v, caseSensitive: %v, useColor %v, useNumber %v, filenameOnly %v, force %v",
		g.ignoreBinary, g.caseSensitive, g.useColor, g.useNumber, g.filenameOnly, g.force)
}

func (g grepp) Run() {
	c := getFileList(g.searchBase, true)

	for filename := range c {
		// fmt.Printf("%s -> %s\n", filename, getMimeType(filename))
		if g.ignoreBinary == true && !isText(filename) {
			continue
		}
		if g.filenameOnly {
			ok, err := checkPatternInFile(filename, g.pattern, !g.caseSensitive)
			if err != nil {
				switch err {
				case errorBufferSizeTooSmall:
					if g.showBufferSizeErrors {
						g.printMinorWarning(fmt.Sprintf("%s : %s", filename, err.Error()))
					} else {
						g.bufferSizeErrorsC++
					}
				default:
					fmt.Fprintf(g.Stderr, "%s", err)
				}
			} else if ok {
				fmt.Fprintf(g.Stdout, "%s%s\n", color(ansi.Magenta, filename, g.useColor), colorReset(g.useColor))
			}
		} else {
			ok, err := checkPatternInFile(filename, g.pattern, !g.caseSensitive)
			if err != nil {
				switch err {
				case errorBufferSizeTooSmall:
					if g.showBufferSizeErrors {
						g.printMinorWarning(fmt.Sprintf("%s : %s", filename, err.Error()))
					} else {
						g.bufferSizeErrorsC++
					}
				default:
					fmt.Fprintf(g.Stderr, "%s", err)
				}
			} else if ok {
				var tmpFile *os.File
				var err error
				if g.force {
					tmpFile, err = ioutil.TempFile("", filepath.Base(filename)+"-")
					defer tmpFile.Close()
					if err != nil {
						l.Error.Println("cannot open ", tmpFile)
						l.Error.Fatal(err)
					}
					l.Debug.Printf("tmpFile: %v", tmpFile.Name())
				}
				for d := range searchInFile(filename, g.pattern, !g.caseSensitive) {
					if len(d.match) == 0 {
						if g.context > 0 {
							g.printLineContext(d)
						}
					} else {
						g.printLineMatch(d)
					}
					if g.force {
						if len(d.match) == 0 {
							tmpFile.WriteString(d.line + "\n")
						} else {
							g.writeLineMatch(tmpFile, d)
						}
					}
				}
				if g.force {
					tmpFile.Close()
					err = copyFileContents(tmpFile.Name(), filename)
					if err != nil {
						l.Warning.Printf("Couldn't update file: %s. '%s'\n", filename, err)
					}
				}
			}
		}
	}

	if g.bufferSizeErrorsC > 0 {
		fmt.Fprintf(g.Stderr, "WARNING: %s found %d times", errorBufferSizeTooSmall, g.bufferSizeErrorsC)
	}
}

func (g *grepp) SetStderr(w io.Writer) {
	l.Warning.SetOutput(w)
	l.Error.SetOutput(w)
	g.Stderr = w
}

func (g *grepp) SetStdout(w io.Writer) {
	l.Info.SetOutput(w)
	g.Stdout = w
}

func synopsis() {
	synopsis := `grepp <pattern> [<location>] [-r <replace pattern> [-f]]
      [-I] [-c] [-n] [-l] [--color]
      [--buffer <size>] [--show-buffer-errors|--sbe]
      [--debug | --trace]

# not available yet
[-C <lines of context>] [--fp] [--name <file pattern>]
[--spacing] [--ignore <file pattern>]

grepp -h # show this help
man grepp # show manpage`
	fmt.Fprintln(os.Stderr, synopsis)
}

func main() {
	l.LogInit(ioutil.Discard, ioutil.Discard, os.Stdout, os.Stderr, os.Stderr)
	l.Debug.Printf("args: %s", os.Args[1:])

	var noPager bool
	var debug, trace bool
	g := grepp{}
	opt := getoptions.GetOptions()
	opt.Flag("h")       // Help
	opt.Flag("version") // version info
	opt.FlagVar(&g.ignoreBinary, "I")
	opt.FlagVar(&g.caseSensitive, "c")
	opt.FlagVar(&g.useColor, "color")
	opt.FlagVar(&g.useNumber, "n")
	opt.FlagVar(&g.filenameOnly, "l")
	opt.StringVar(&g.replace, "r")
	opt.FlagVar(&g.force, "f")
	opt.IntVar(&g.context, "C")
	opt.IntVar(&bufferSize, "buffer")
	opt.FlagVar(&g.showBufferSizeErrors, "show-buffer-errors", "sbe")
	opt.FlagVar(&noPager, "no-pager")
	opt.FlagVar(&debug, "debug") // debug logging
	opt.FlagVar(&trace, "trace") // trace logging
	// "fp"      // fullPath - Used to show the file full path instead of the relative to the current dir.
	// "name"    // filePattern - Use to further filter the search to files matching that pattern.
	// "ignore"  // ignoreFilePattern - Use to further filter the search to files not matching that pattern.
	// "spacing" // keepSpacing - Do not remove initial spacing.
	// "no-page" // Don't use pager for output
	remaining, err := opt.Parse(os.Args[1:])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if opt.Option["h"] != nil && opt.Option["h"].(bool) {
		synopsis()
		os.Exit(1)
	}

	if opt.Option["version"] != nil && opt.Option["version"].(bool) {
		version := semver.Version{Major: 0, Minor: 9, Patch: 0, PreReleaseLabel: "dev"}
		fmt.Println(version)
		os.Exit(1)
	}

	// Check if stdout is pipe p or device D
	statStdout, _ := os.Stdout.Stat()
	stdoutIsDevice := (statStdout.Mode() & os.ModeDevice) != 0

	if debug || trace {
		l.Debug.SetOutput(os.Stderr)
	}
	if trace {
		l.Trace.SetOutput(os.Stderr)
	}

	l.Debug.Printf("stats Stdout: %s, is device: %v", statStdout.Mode(), stdoutIsDevice)

	if len(remaining) < 1 {
		l.Error.Fatal("Missing pattern!")
	}
	if len(remaining) < 2 {
		g.searchBase = "."
	} else {
		g.searchBase = remaining[1]
	}
	searchBaseInfo, err := os.Stat(g.searchBase)
	if err != nil {
		l.Error.Println("cannot stat", g.searchBase)
		l.Error.Fatal(err)
	}
	if searchBaseInfo.IsDir() {
		g.showFile = true
	} else {
		g.showFile = false
	}

	g.pattern = remaining[0]

	l.Debug.Printf("pattern: %s, searchBase: %s, replace: %s", g.pattern, g.searchBase, g.replace)
	l.Debug.Printf(fmt.Sprintln(g))

	g.useColor = !g.useColor

	if !noPager && stdoutIsDevice {
		l.Debug.Println("runInPager")
		runInPager.Command(&g)
		os.Exit(0)
	} else if noPager && stdoutIsDevice {
		g.Stdout = os.Stdout
		g.Stderr = os.Stderr
		g.Run()
	} else {
		g.useColor = false
		g.Stdout = os.Stdout
		g.Stderr = os.Stderr
		g.Run()
	}
}
