package main

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

		{"opt", "singleDash", []string{}, ""},
		{"--opt", "singleDash", []string{"opt"}, ""},
		{"--opt=arg", "singleDash", []string{"opt"}, "arg"},
		{"-opt", "singleDash", []string{"o"}, "pt"},
		{"-opt=arg", "singleDash", []string{"o"}, "pt=arg"},

		{"opt", "normal", []string{}, ""},
		{"--opt", "normal", []string{"opt"}, ""},
		{"--opt=arg", "normal", []string{"opt"}, "arg"},
		{"-opt", "normal", []string{"opt"}, ""},
		{"-opt=arg", "normal", []string{"opt"}, "arg"},
	}
	for _, c := range cases {
		options, argument := isOption(c.in, c.mode)
		if !reflect.DeepEqual(options, c.options) || argument != c.argument {
			t.Errorf("isOption(%q, %q) == (%q, %q), want (%q, %q)",
				c.in, c.mode, options, argument, c.options, c.argument)
		}
	}
}

func TestGetOptLong(t *testing.T) {
	cases := []struct {
		in         []string
		mode       string
		definition map[string]optDef
		options    map[string]interface{}
	}{
		{[]string{},
			"bundling",
			map[string]optDef{"flag": optDef{"", nil}},
			map[string]interface{}{},
		},
		{[]string{"--flag"},
			"bundling",
			map[string]optDef{"flag": optDef{"", nil}},
			map[string]interface{}{"flag": true},
		},
		{[]string{"--int=123"},
			"bundling",
			map[string]optDef{"int": optDef{"=i", nil}},
			map[string]interface{}{"int": 123},
		},
		{[]string{"--int 123"},
			"bundling",
			map[string]optDef{"int": optDef{"=i", nil}},
			map[string]interface{}{"int": 123},
		},
	}
	for _, c := range cases {
		options, err := getOptLong(c.in, c.mode, c.definition)
		if !reflect.DeepEqual(options, c.options) {
			t.Errorf("getOptLong(%q, %q, %q) == (%v, %q), want (%v, %q)",
				c.in, c.mode, c.definition, options, err, c.options, nil)
		}
	}
}
