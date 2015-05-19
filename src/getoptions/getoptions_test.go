package getoptions

import (
	"reflect"
	"testing"
)

func TestIsOption(t *testing.T) {
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
	}{
		{[]string{},
			"bundling",
			OptDef{"flag": {"", nil}},
			Options{},
		},
		{[]string{"--flag"},
			"bundling",
			OptDef{"flag": {"", nil}},
			Options{"flag": true},
		},
	}
	for _, c := range cases {
		options, err := GetOptLong(c.in, c.mode, c.definition)
		if !reflect.DeepEqual(options, c.options) {
			t.Errorf("getOptLong(%q, %q, %v) == (%v, %v), want (%v, %v)",
				c.in, c.mode, c.definition, options, err, c.options, nil)
		}
	}
}

func TestGetOptInt(t *testing.T) {
	cases := []struct {
		in         []string
		mode       string
		definition OptDef
		options    Options
	}{
		{[]string{"--int=123"},
			"bundling",
			OptDef{"int": {"=i", nil}},
			Options{"int": 123},
		},
		{[]string{"--int", "123"},
			"bundling",
			OptDef{"int": {"=i", nil}},
			Options{"int": int(123)},
		},
	}
	for _, c := range cases {
		options, err := GetOptLong(c.in, c.mode, c.definition)
		if !reflect.DeepEqual(options, c.options) {
			t.Errorf("getOptLong(%q, %q, %q) == (%v, %v), want (%v, %v)",
				c.in, c.mode, c.definition, options, err, c.options, nil)
		}
	}
}

func TestGetOptLong(t *testing.T) {
	cases := []struct {
		in         []string
		mode       string
		definition OptDef
		options    Options
	}{
		{[]string{"--string", "hello", "--int", "123", "--flag"},
			"bundling",
			OptDef{
				"flag":   {"", nil},
				"int":    {"=i", nil},
				"string": {"=s", nil},
			},
			Options{"int": 123, "string": "hello", "flag": true},
		},
	}
	for _, c := range cases {
		options, err := GetOptLong(c.in, c.mode, c.definition)
		if !reflect.DeepEqual(options, c.options) {
			t.Errorf("getOptLong(%q, %q, %v) == (%v, %v), want (%v, %v)",
				c.in, c.mode, c.definition, options, err, c.options, nil)
		}
	}
}
