package helpers

import (
	"strings"
)

// FindString find specified value in a list of strings
// return index of found item or -1 if value is not in the collection
func FindString(collection []string, value string) int {
	for i := 0; i < len(collection); i++ {
		if collection[i] == value {
			return i
		}
	}
	return -1
}

// FindStringNC find specified value in a list of string ignoring case of the strings
// return index of found item or -1 if value is not in the collection
func FindStringNC(collection []string, value string) int {
	value = strings.ToLower(value)
	for i := 0; i < len(collection); i++ {
		if strings.ToLower(collection[i]) == value {
			return i
		}
	}
	return -1
}

// FindStringIf find a value in a collection of strings that satisfy specified predicate
// return index of found item or -1 if no value found
func FindStringIf(collection []string, pred func(string) bool) int {
	for i := 0; i < len(collection); i++ {
		if pred(collection[i]) {
			return i
		}
	}
	return -1
}

// ContainsString check whether a collection contains a value or not
func ContainsString(collection []string, value string) bool {
	return FindString(collection, value) != -1
}

// ContainsStringNC check whether a collection contains a value or not ignoring the case of the values
func ContainsStringNC(collection []string, value string) bool {
	return FindStringNC(collection, value) != -1
}

// ContainsStringIf check whether a collection contains a value or not
func ContainsStringIf(collection []string, pred func(string) bool) bool {
	return FindStringIf(collection, pred) != -1
}
