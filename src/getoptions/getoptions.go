/*
package getoptions - Go option parser based on Perl’s Getopt::Long.
*/
package getoptions

import (
	"fmt"
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

/*
func GetOptLong -
*/
func GetOptLong(args []string,
	mode string,
	definition OptDef) (Options, error) {
	options := Options{}
	fmt.Printf("GetOptLong args: %v\n", args)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		fmt.Printf("GetOptLong input arg: %d, %s\n", i, arg)
		if match, argument := isOption(arg, mode); len(match) > 0 {
			fmt.Printf("GetOptLong match: %v, argument: %v\n", match, argument)
			if _, ok := definition[match[0]]; ok {
				fmt.Printf("GetOptLong found\n")
				switch definition[match[0]].Spec {
				case "":
					options[match[0]] = true
				case "!":
					options[match[0]] = false
				case "=s":
					if argument != "" {
						options[match[0]] = argument
					} else {
						i++
						options[match[0]] = args[i]
					}
				case "=i":
					if argument != "" {
						if iArg, err := strconv.Atoi(argument); err != nil {
							panic(fmt.Sprintf("Can't convert string to int: %q", err))
						} else {
							options[match[0]] = iArg
						}
					} else {
						i++
						if iArg, err := strconv.Atoi(args[i]); err != nil {
							panic(fmt.Sprintf("Can't convert string to int: %q", err))
						} else {
							options[match[0]] = iArg
						}
					}
				}
			}
		}
	}
	fmt.Printf("GetOptLong options: %v\n", options)
	return options, nil
}
