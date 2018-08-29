package schema

import (
	"regexp"
)

var nameFormat *regexp.Regexp
var dimFormat *regexp.Regexp

func checkNameFormat(s string) bool {
	if nameFormat == nil {
		nameFormat = regexp.MustCompile("^[a-z_][0-9a-z_]*\\z")
	}
	return nameFormat.MatchString(s)
}

func checkDimFormat(s string) bool {
	if dimFormat == nil {
		dimFormat = regexp.MustCompile("^[a-z_][0-9a-z_]*[+?*]?\\z")
	}
	return dimFormat.MatchString(s)
}
