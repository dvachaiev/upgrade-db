package db

import (
	"testing"
)

func TestString(t *testing.T) {
	var tests = []struct {
		seq      []int
		expected string
	}{
		{[]int{128}, "128"},                 // single value
		{[]int{128, 0}, "128.0"},            // ends with 0
		{[]int{0, 128}, "0.128"},            // starts with 0
		{[]int{1, 2, 3, 4, 5}, "1.2.3.4.5"}, // with 5 elements
		{[]int{0}, "0"},                     // single zero
		{[]int{0, 0, 0, 0}, "0.0.0.0"},      // several zeros
	}
	for _, test := range tests {
		if ver := (Version{test.seq}); ver.String() != test.expected {
			t.Error("Failed:", ver.String(), "!=", test.expected)
		}
	}
}

func TestLess(t *testing.T) {
	var tests = []struct {
		left, right []int
		expected    bool
	}{
		// Equal
		{[]int{1, 1, 1}, []int{1, 1, 1}, true},
		{[]int{1, 0}, []int{1, 0, 0}, true},
		{[]int{1, 0, 0}, []int{1, 0}, true},

		// Left is less
		{[]int{1, 1, 1}, []int{1, 1, 2}, true},
		{[]int{1, 1, 1}, []int{1, 2, 1}, true},
		{[]int{1, 1, 1}, []int{2, 1, 1}, true},
		{[]int{1, 1, 1}, []int{1, 2}, true},
		{[]int{1, 1}, []int{1, 1, 1}, true},

		// Right is less
		{[]int{1, 1, 2}, []int{1, 1, 1}, false},
		{[]int{1, 2, 1}, []int{1, 1, 1}, false},
		{[]int{2, 1, 1}, []int{1, 1, 1}, false},
		{[]int{1, 1, 1}, []int{1, 1}, false},
		{[]int{1, 2}, []int{1, 1, 1}, false},
	}
	for _, test := range tests {
		lver := Version{test.left}
		rver := Version{test.right}
		if res := lver.Less(rver); res != test.expected {
			t.Error("Failed:", lver, "<", rver, "!=", test.expected)
		}
	}
}
