package main

import (
	l "github.com/davidgamba/grepp/logging"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func TestMain(m *testing.M) {
	// l.LogInit(os.Stderr, os.Stderr, os.Stdout, os.Stderr, os.Stderr)
	l.LogInit(ioutil.Discard, ioutil.Discard, os.Stdout, os.Stderr, os.Stderr)
	os.Exit(m.Run())
}

func TestGetFileList(t *testing.T) {
	cases := []struct {
		file      string
		ignoreDir bool
		result    []string
	}{
		{"./test_tree", false, []string{
			"./test_tree",
			"test_tree/.A",
			"test_tree/.A/b",
			"test_tree/.A/b/C",
			"test_tree/.A/b/C/d",
			"test_tree/.A/b/C/d/E",
			"test_tree/.a",
			"test_tree/.a/B",
			"test_tree/.a/B/c",
			"test_tree/.a/B/c/D",
			"test_tree/.a/B/c/D/e",
			"test_tree/.svn",
			"test_tree/.svn/E",
			"test_tree/.svn/e",
			"test_tree/A",
			"test_tree/A/b",
			"test_tree/A/b/C",
			"test_tree/A/b/C/d",
			"test_tree/A/b/C/d/E",
			"test_tree/a",
			"test_tree/a/B",
			"test_tree/a/B/c",
			"test_tree/a/B/c/D",
			"test_tree/a/B/c/D/e"},
		},
		{"./test_tree", true, []string{
			"test_tree/.A/b/C/d/E",
			"test_tree/.a/B/c/D/e",
			"test_tree/.svn/E",
			"test_tree/.svn/e",
			"test_tree/A/b/C/d/E",
			"test_tree/a/B/c/D/e",
		},
		},
	}
	for _, c := range cases {
		d := getFileList(c.file, c.ignoreDir)
		tree := []string{}
		for file := range d {
			tree = append(tree, file)
		}
		if !reflect.DeepEqual(tree, c.result) {
			t.Errorf("tree %q != %q", c.result, tree)
		}
	}
}

func TestCheckPatternInFile(t *testing.T) {
	cases := []struct {
		file       string
		pattern    string
		ignoreCase bool
		result     bool
	}{
		{"test_tree/A/b/C/d/E", "loreM", true, true},
		{"test_tree/A/b/C/d/E", "loreM", false, false},
		{"test_tree/A/b/C/d/E", "test", false, false},
		{"test_tree/A/b/C/d/E", "test", true, false},
	}
	for _, c := range cases {
		r := checkPatternInFile(c.file, c.pattern, c.ignoreCase)
		if r != c.result {
			t.Errorf("checkPatternInFile(%q, %q, %v) == (%v), want (%v)",
				c.file, c.pattern, c.ignoreCase, r, c.result)
		}
	}
}

func TestGetRegex(t *testing.T) {
	cases := []struct {
		pattern    string
		ignoreCase bool
		line       string
		before     string
		match      string
		after      string
	}{
		{"pattern", true, "before pattern after", "before ", "pattern", " after"},
		{"(pattern)+", true, "before patternpattern after", "before ", "patternpattern", " after"},
		{"(pattern)(capture)(groups)", true, "before patterncapturegroups after", "before ", "patterncapturegroups", " after"},
	}
	for _, c := range cases {
		re, reEnd := getRegex(c.pattern, c.ignoreCase)
		match := re.FindAllStringSubmatch(c.line, -1)
		remainder := reEnd.FindStringSubmatch(c.line)
		if match[0][1] != c.before || match[0][2] != c.match || remainder[len(remainder)-1] != c.after {
			t.Errorf("TestGetRegex: expected %q, %q, %q | result match: %q, remainder: %q", c.before, c.match, c.after, match, remainder)
		}
	}
}
