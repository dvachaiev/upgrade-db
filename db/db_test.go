package db

import (
	"testing"
)

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
			t.Error("Failed:", lver, "<", rver, "==", test.expected)
		}
	}
}
