package funson

import (
	"reflect"
	"testing"
)

func TestIsSliceFunc(t *testing.T) {
	tests := []struct {
		name  string
		input []interface{}
		want  bool
	}{
		{
			name:  "function call",
			input: []interface{}{"!add", 1, 2},
			want:  true,
		},
		{
			name:  "function call single arg",
			input: []interface{}{"!noop"},
			want:  true,
		},
		{
			name:  "numbers array",
			input: []interface{}{1, 2, 3},
			want:  false,
		},
		{
			name:  "double bang",
			input: []interface{}{"!!notFunc"},
			want:  false,
		},
		{
			name:  "bang only",
			input: []interface{}{"!"},
			want:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isSliceFunc(tc.input)
			if got != tc.want {
				t.Errorf("isSliceFunc(%v) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestSliceFunc(t *testing.T) {
	tests := []struct {
		name  string
		input []interface{}
		fname string
		args  []interface{}
	}{
		{
			name:  "call add",
			input: []interface{}{"!add", 1, 2},
			fname: "add",
			args:  []interface{}{1, 2},
		},
		{
			name:  "call noop",
			input: []interface{}{"!noop"},
			fname: "noop",
			args:  []interface{}{},
		},
		{
			name:  "not func numbers",
			input: []interface{}{1, 2, 3},
			fname: "",
			args:  nil,
		},
		{
			name:  "not func double bang",
			input: []interface{}{"!!no"},
			fname: "",
			args:  nil,
		},
		{
			name:  "bang only",
			input: []interface{}{"!"},
			fname: "",
			args:  nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fn, a := sliceFunc(tc.input)
			if fn != tc.fname || !reflect.DeepEqual(a, tc.args) {
				t.Errorf("sliceFunc(%v) = (%q, %v), want (%q, %v)", tc.input, fn, a, tc.fname, tc.args)
			}
		})
	}
}
