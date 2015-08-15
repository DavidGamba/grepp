package getoptions

import (
	l "github.com/davidgamba/grepp/logging"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func TestIsOption(t *testing.T) {
	l.LogInit(ioutil.Discard, ioutil.Discard, os.Stdout, os.Stderr, os.Stderr)
	cases := []struct {
		in       string
		mode     string
		options  []string
		argument string
	}{
		{"opt", "bundling", []string{}, ""},
		{"--opt", "bundling", []string{"opt"}, ""},
		{"--opt=arg", "bundling", []string{"opt"}, "arg"},
		{"-opt", "bundling", []string{"o", "p", "t"}, ""},
		{"-opt=arg", "bundling", []string{"o", "p", "t"}, "arg"},
		{"-", "bundling", []string{"-"}, ""},
		{"--", "bundling", []string{"--"}, ""},

		{"opt", "singleDash", []string{}, ""},
		{"--opt", "singleDash", []string{"opt"}, ""},
		{"--opt=arg", "singleDash", []string{"opt"}, "arg"},
		{"-opt", "singleDash", []string{"o"}, "pt"},
		{"-", "singleDash", []string{"-"}, ""},
		{"--", "singleDash", []string{"--"}, ""},

		{"opt", "normal", []string{}, ""},
		{"--opt", "normal", []string{"opt"}, ""},
		{"--opt=arg", "normal", []string{"opt"}, "arg"},
		{"-opt", "normal", []string{"opt"}, ""},
		{"-", "normal", []string{"-"}, ""},
		{"--", "normal", []string{"--"}, ""},
	}
	for _, c := range cases {
		options, argument := isOption(c.in, c.mode)
		if !reflect.DeepEqual(options, c.options) || argument != c.argument {
			t.Errorf("isOption(%q, %q) == (%q, %q), want (%q, %q)",
				c.in, c.mode, options, argument, c.options, c.argument)
		}
	}
}

func TestGetOptFlag(t *testing.T) {
	cases := []struct {
		in         []string
		mode       string
		definition OptDef
		options    Options
		remaining  []string
	}{
		{[]string{},
			"bundling",
			OptDef{"flag": {"", false}},
			Options{"flag": false},
			[]string{},
		},
		{[]string{"--flag"},
			"bundling",
			OptDef{"flag": {"", false}},
			Options{"flag": true},
			[]string{},
		},
	}
	for _, c := range cases {
		options, remaining := GetOptLong(c.in, c.mode, c.definition)
		if !reflect.DeepEqual(options, c.options) {
			t.Errorf("getOptLong(%q, %q, %v) == (%v, %v), want (%v, %v)",
				c.in, c.mode, c.definition, options, remaining, c.options, c.remaining)
		}
	}
}

func TestGetOptInt(t *testing.T) {
	cases := []struct {
		in         []string
		mode       string
		definition OptDef
		options    Options
		remaining  []string
	}{
		{[]string{"--int=123"},
			"bundling",
			OptDef{"int": {"=i", 0}},
			Options{"int": 123},
			[]string{},
		},
		{[]string{"--int", "123"},
			"bundling",
			OptDef{"int": {"=i", 0}},
			Options{"int": int(123)},
			[]string{},
		},
	}
	for _, c := range cases {
		options, err := GetOptLong(c.in, c.mode, c.definition)
		if !reflect.DeepEqual(options, c.options) {
			t.Errorf("getOptLong(%q, %q, %q) == (%v, %v), want (%v, %v)",
				c.in, c.mode, c.definition, options, err, c.options, c.remaining)
		}
	}
}

func TestGetOptLong(t *testing.T) {
	cases := []struct {
		in         []string
		mode       string
		definition OptDef
		options    Options
		remaining  []string
	}{
		{[]string{"--string", "hello", "--int", "123", "--flag"},
			"bundling",
			OptDef{
				"flag":   {"", false},
				"int":    {"=i", 0},
				"string": {"=s", ""},
			},
			Options{"int": 123, "string": "hello", "flag": true},
			nil,
		},
		{[]string{"t1", "--string", "hello", "t2", "--int", "123", "t3", "--flag", "t4"},
			"bundling",
			OptDef{
				"flag":   {"", false},
				"int":    {"=i", 0},
				"string": {"=s", ""},
			},
			Options{"int": 123, "string": "hello", "flag": true},
			[]string{"t1", "t2", "t3", "t4"},
		},
	}
	for _, c := range cases {
		options, remaining := GetOptLong(c.in, c.mode, c.definition)
		if !reflect.DeepEqual(options, c.options) {
			t.Errorf("getOptLong(%q, %q, %v) == (%v, %v), want (%v, %v)",
				c.in, c.mode, c.definition, options, remaining, c.options, c.remaining)
		}
		if !reflect.DeepEqual(remaining, c.remaining) {
			t.Errorf("getOptLong(%q, %q, %v) == (%v, %v), want (%v, %v)",
				c.in, c.mode, c.definition, options, remaining, c.options, c.remaining)
		}
	}
}
