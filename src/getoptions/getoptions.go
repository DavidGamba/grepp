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

func isOption(s string, mode string) ([]string, string) {
	match := isOptionRegex.FindStringSubmatch(s)
	if len(match) > 0 {
		// check long option
		if match[1] == "--" {
			return []string{match[2]}, isOptionRegexEquals.ReplaceAllString(match[3], "")
		} else {
			switch mode {
			case "bundling":
				return strings.Split(match[2], ""), isOptionRegexEquals.ReplaceAllString(match[3], "")
			case "singleDash":
				return []string{strings.Split(match[2], "")[0]}, strings.Join(strings.Split(match[2], "")[1:], "") + match[3]
			default:
				return []string{match[2]}, isOptionRegexEquals.ReplaceAllString(match[3], "")
			}
		}
	}
	return []string{}, ""
}

type OptDef struct {
	spec  string
	value interface{}
}

func GetOptLong(args []string,
	mode string,
	definition map[string]OptDef) (map[string]interface{}, error) {
	options := map[string]interface{}{}
	for i, arg := range args {
		fmt.Printf("input arg: %d, %s\n", i, arg)
		if match, argument := isOption(arg, mode); len(match) > 0 {
			fmt.Printf("match: %v, argument: %v\n", match, argument)
			if _, ok := definition[match[0]]; ok {
				fmt.Printf("found: '%v'\n", ok)
				switch definition[match[0]].spec {
				case "":
					options[match[0]] = true
				case "=i":
					if argument != "" {
						if i, err := strconv.Atoi(argument); err != nil {
							panic(fmt.Sprintf("Can't convert string to int: %q", err))
						} else {
							options[match[0]] = i
						}
					} else {
						//TODO: Get next arg
					}
				}
			}
		}
	}
	return options, nil
}
