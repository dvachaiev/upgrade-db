package db

import (
	"strconv"
	"testing"

	"github.com/matryer/is"
)

func TestString(t *testing.T) {
	is := is.New(t)

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
		ver := (Version{test.seq})
		is.Equal(ver.String(), test.expected)
	}
}

func TestLess(t *testing.T) {
	is := is.New(t)

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
		is.Equal(lver.Less(rver), test.expected)
	}
}

func TestVersion_IsZero(t *testing.T) {
	type fields struct {
		seq []int
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{name: "Nil slice", fields: fields{seq: nil}, want: true},
		{name: "Empty slice", fields: fields{seq: []int{}}, want: true},
		{name: "Non-empty slice", fields: fields{seq: []int{1, 2, 3}}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := Version{
				seq: tt.fields.seq,
			}
			if got := v.IsZero(); got != tt.want {
				t.Errorf("Version.IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersion_Equal(t *testing.T) {
	tests := []struct {
		left, right []int
		expected    bool
	}{
		// Equal
		{[]int{1, 1, 1}, []int{1, 1, 1}, true},
		{[]int{1, 0}, []int{1, 0, 0}, true},
		{[]int{1, 0, 0}, []int{1, 0}, true},

		// Left is less
		{[]int{1, 1, 1}, []int{1, 1, 2}, false},
		{[]int{1, 1, 1}, []int{1, 2, 1}, false},
		{[]int{1, 1, 1}, []int{2, 1, 1}, false},
		{[]int{1, 1, 1}, []int{1, 2}, false},
		{[]int{1, 1}, []int{1, 1, 1}, false},

		// Right is less
		{[]int{1, 1, 2}, []int{1, 1, 1}, false},
		{[]int{1, 2, 1}, []int{1, 1, 1}, false},
		{[]int{2, 1, 1}, []int{1, 1, 1}, false},
		{[]int{1, 1, 1}, []int{1, 1}, false},
		{[]int{1, 2}, []int{1, 1, 1}, false},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			is := is.New(t)

			lver := Version{tt.left}
			rver := Version{tt.right}

			is.Equal(lver.Equal(rver), tt.expected)
		})
	}
}

func TestParseVersion(t *testing.T) {
	is := is.NewRelaxed(t)

	var empty []int

	var tests = []struct {
		input    string
		expected []int
		isError  bool
	}{
		{"128", []int{128}, false},                 // single value
		{"128.0", []int{128, 0}, false},            // ends with 0
		{"0.128", []int{0, 128}, false},            // starts with 0
		{"1.2.3.4.5", []int{1, 2, 3, 4, 5}, false}, // with 5 elements
		{"0", []int{0}, false},                     // single zero
		{"0.0.0.0", []int{0, 0, 0, 0}, false},      // several zeros

		{"1.2.3.", empty, true},  // empty value at the end
		{".1.2.3.", empty, true}, // empty value at the beginning
		{"1..3.", empty, true},   // empty value in the middle
		{"1.a.3.", empty, true},  // alpha in the middle
	}

	for _, test := range tests {
		ver, err := ParseVersion(test.input)
		is.Equal(ver.seq, test.expected)

		if test.isError {
			is.True(err != nil)
		} else {
			is.NoErr(err)
		}
	}
}
