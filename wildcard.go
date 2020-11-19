package helpers

import (
	"regexp"
	"strings"
)

// IsWildcard Check if an string contains wildcard syntax characters
func IsWildcard(s string) bool { return strings.IndexAny(s, "*?") != -1 }

// WildcardToRegexp Compile wildcard to a regexp pattern
func WildcardToRegexp(s string) string {
	pattern := ""
	for {
		i := strings.IndexAny(s, "?*")
		if i == -1 {
			pattern += regexp.QuoteMeta(s)
			break
		} else {
			pattern += regexp.QuoteMeta(s[:i])
			if s[i] == '?' {
				pattern += "."
			} else {
				pattern += ".*"
			}
			s = s[i+1:]
		}
	}
	return pattern
}
