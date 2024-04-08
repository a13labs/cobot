package algo

// This file provides text matching algorithms.

// Function that recieves a list of strings and a matching pattern, and returns the strings that match the pattern.
// the pattern can contain the '*' character, which is a wildcard that matches any character.
func MatchAll(pattern string, strings []string) []string {
	var matches []string
	for _, str := range strings {
		if Match(pattern, str) {
			matches = append(matches, str)
		}
	}
	return matches
}

// Function that recieves a pattern and a string, and returns true if the string matches the pattern.
// The pattern can contain the '*' character, which is a wildcard that matches any character.
func Match(pattern string, str string) bool {
	if pattern == "" {
		return true
	}
	if pattern == "*" {
		return true
	}
	if pattern == str {
		return true
	}
	if pattern[0] == '*' {
		for i := 0; i <= len(str); i++ {
			if Match(pattern[1:], str[i:]) {
				return true
			}
		}
		return false
	}
	if str == "" {
		return false
	}
	if pattern[0] == str[0] {
		return Match(pattern[1:], str[1:])
	}
	return false
}
