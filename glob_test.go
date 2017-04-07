package main

import "testing"

var globTests = []struct {
	pattern  string
	in       string
	expected bool
}{
	{"", "", true},
	{"", "file.txt", false},
	{"file.txt", "file.txt", true},
	{"*", "file.txt", true}, // matches everything
	{"*", "", true},         // matches everything
	{"*beta", "", false},
	{"*beta", "beta", true},
	{"*beta", "alphabeta", true},

	{"beta*", "betaalpha", true},
	{"beta*", "", false},
	{"beta*", "beta", true},
	{"beta*", "alphabeta", false},

	{"*beta*", "", false},
	{"*beta*", "alpha", false},
	{"*beta*", "beta", true},
	{"*beta*", "alphabeta", true},
	{"*beta*", "betagamma", true},
	{"*beta*", "alphabetagamma", true},
	{"*beta*", "alpha/beta/gamma", true},
}

func TestGlobMatch(t *testing.T) {
	for _, tt := range globTests {
		actual := globMatch(tt.pattern, tt.in)
		if actual != tt.expected {
			t.Errorf("patternMatch(\"%s\", \"%s\") => %t, want %t", tt.pattern, tt.in, actual, tt.expected)
		}
	}
}
