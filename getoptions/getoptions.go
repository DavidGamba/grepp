/*
package getoptions - Go option parser based on Perl’s Getopt::Long.
*/
package getoptions

import (
	l "github.com/davidgamba/grepp/logging"
	"regexp"
	"strconv"
	"strings"
)

var isOptionRegex = regexp.MustCompile(`^(--?)([^=]+)(.*?)$`)
var isOptionRegexEquals = regexp.MustCompile(`^=`)

/*
func isOption - Check if the given string is an option (starts with - or --).
Return the option(s) without the starting dash and an argument if the string contained one.
The behaviour changes depending on the mode: normal, bundling or SingleDash.
Also, handle the single dash '-' and double dash '--' especial options.
*/
func isOption(s string, mode string) (options []string, argument string) {
	// Handle especial cases
	if s == "--" {
		return []string{"--"}, ""
	} else if s == "-" {
		return []string{"-"}, ""
	}

	match := isOptionRegex.FindStringSubmatch(s)
	if len(match) > 0 {
		// check long option
		if match[1] == "--" {
			options = []string{match[2]}
			argument = isOptionRegexEquals.ReplaceAllString(match[3], "")
			return
		} else {
			switch mode {
			case "bundling":
				options = strings.Split(match[2], "")
				argument = isOptionRegexEquals.ReplaceAllString(match[3], "")
			case "singleDash":
				options = []string{strings.Split(match[2], "")[0]}
				argument = strings.Join(strings.Split(match[2], "")[1:], "") + match[3]
			default:
				options = []string{match[2]}
				argument = isOptionRegexEquals.ReplaceAllString(match[3], "")
			}
			return
		}
	}
	return []string{}, ""
}

// type OptDef - Definition "Spec" and default "Value".
type OptDef map[string]struct {
	Spec  string
	Value interface{}
}

type Options map[string]interface{}

func handleOption(definition OptDef,
	alias string,
	argument string,
	args []string,
	_options *Options,
	i *int) {

	options := *_options

	switch definition[alias].Spec {
	case "":
		if val, ok := options[alias]; ok {
			options[alias] = !val.(bool)
		} else {
			options[alias] = true
		}
	case "!":
		options[alias] = false
	case "=s":
		if argument != "" {
			options[alias] = argument
		} else {
			*i++
			options[alias] = args[*i]
		}
	case "=i":
		if argument != "" {
			if iArg, err := strconv.Atoi(argument); err != nil {
				l.Error.Panicf("Can't convert string to int: %q", err)
			} else {
				options[alias] = iArg
			}
		} else {
			*i++
			if iArg, err := strconv.Atoi(args[*i]); err != nil {
				l.Error.Panicf("Can't convert string to int: %q", err)
			} else {
				options[alias] = iArg
			}
		}
	}
}

// func setOptionDefaults - Read definition and set the default option values
func setOptionDefaults(definition OptDef, _options *Options) {
	options := *_options
	for k, v := range definition {
		options[k] = v.Value
	}
}

/*
func GetOptLong -
*/
func GetOptLong(args []string,
	mode string,
	definition OptDef) (Options, []string) {

	options := Options{}
	var remaining []string

	l.Debug.Printf("GetOptLong args: %v\n", args)
	l.Debug.Printf("GetOptLong definition: %v\n", definition)

	setOptionDefaults(definition, &options)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		l.Debug.Printf("GetOptLong input arg: %s\n", arg)
		if match, argument := isOption(arg, mode); len(match) > 0 {
			l.Debug.Printf("GetOptLong match: %v, argument: %v\n", match, argument)
			// Check for termination: '--'
			if match[0] == "--" {
				l.Debug.Printf("GetOptLong -- found\n")
				remaining = append(remaining, args[i+1:]...)
				return options, remaining
			}
			if _, ok := definition[match[0]]; ok {
				l.Debug.Printf("GetOptLong found\n")
				handleOption(definition, match[0], argument, args, &options, &i)
			} else {
				// TODO: Handle invalid options
				remaining = append(remaining, arg)
			}
		} else {
			remaining = append(remaining, arg)
		}
	}
	l.Debug.Printf("GetOptLong options: %v, remaining: %v\n", options, remaining)
	return options, remaining
}
