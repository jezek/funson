package funson

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestFun(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    interface{}
		wantErr error
	}{
		{
			name:    "function slice",
			input:   []interface{}{"!add", float64(1), float64(2)},
			want:    float64(3),
			wantErr: nil,
		},
		{
			name:    "plain values",
			input:   []interface{}{float64(1), []interface{}{"!add", float64(2), float64(3)}, float64(4)},
			want:    []interface{}{float64(1), float64(5), float64(4)},
			wantErr: nil,
		},
		{
			name:    "variadic single",
			input:   []interface{}{"!not", true},
			want:    false,
			wantErr: nil,
		},
		{
			name:    "variadic multiple results",
			input:   []interface{}{"!not", true, false},
			want:    Result{false, true},
			wantErr: ErrorMultipleValues{Values: Result{false, true}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Fun(tc.input)
			if !reflect.DeepEqual(got, tc.want) || !reflect.DeepEqual(err, tc.wantErr) {
				t.Fatalf("Fun(%v) = (%v, %v), want (%v, %v)", tc.input, got, err, tc.want, tc.wantErr)
			}
		})
	}
}

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

func TestProcessSliceFunc(t *testing.T) {
	origFunctions := copyMap(functions)
	defer func() { functions = origFunctions }()

	functions = map[string]interface{}{}

	called := false
	if err := AddFun("addTest", func(_ *EnviromentNode, a, b float64) float64 {
		called = true
		return a + b
	}); err != nil {
		t.Fatalf("AddFun(addTest) error = %v", err)
	}

	if err := AddFun("concatTest", func(_ *EnviromentNode, strs ...string) string {
		called = true
		return strings.Join(strs, "")
	}); err != nil {
		t.Fatalf("AddFun(concatTest) error = %v", err)
	}

	if err := AddFun("noopTest", func(_ *EnviromentNode) {
		called = true
	}); err != nil {
		t.Fatalf("AddFun(noopTest) error = %v", err)
	}

	if err := AddFun("resultsTest", func(_ *EnviromentNode, values ...float64) Result {
		results := make(Result, len(values))
		for i, value := range values {
			results[i] = value
		}
		return results
	}); err != nil {
		t.Fatalf("AddFun(resultsTest) error = %v", err)
	}

	if err := AddFun("stringsTest", func(_ *EnviromentNode) Result {
		return Result{"a", "b"}
	}); err != nil {
		t.Fatalf("AddFun(stringsTest) error = %v", err)
	}

	en := (&EnviromentNode{Enviroment{}, nil}).Child(Enviroment{})

	tests := []struct {
		name       string
		fname      string
		args       []interface{}
		want       interface{}
		wantErr    error
		wantCalled bool
	}{
		{"add numbers", "addTest", []interface{}{float64(1), float64(2)}, Result{float64(3)}, nil, true},
		{"extra raw argument", "addTest", []interface{}{float64(1), float64(2), float64(3)}, nil, ErrorTooManyArguments{Name: "addTest", Remaining: 1}, false},
		{"exact expanded arguments", "addTest", []interface{}{[]interface{}{"!resultsTest", float64(1), float64(2)}}, Result{float64(3)}, nil, true},
		{"extra expanded argument", "addTest", []interface{}{[]interface{}{"!resultsTest", float64(1), float64(2), float64(3)}}, nil, ErrorTooManyArguments{Name: "addTest", Remaining: 1}, false},
		{"extra raw and expanded arguments", "addTest", []interface{}{[]interface{}{"!resultsTest", float64(1), float64(2), float64(3)}, float64(4)}, nil, ErrorTooManyArguments{Name: "addTest", Remaining: 2}, false},
		{"no arguments", "noopTest", nil, Result{}, nil, true},
		{"extra argument for zero argument function", "noopTest", []interface{}{float64(1)}, nil, ErrorTooManyArguments{Name: "noopTest", Remaining: 1}, false},
		{"concat", "concatTest", []interface{}{"a", "b", "c"}, Result{"abc"}, nil, true},
		{"concat expanded arguments", "concatTest", []interface{}{[]interface{}{"!stringsTest"}, "c"}, Result{"abc"}, nil, true},
		{"concat no arguments", "concatTest", nil, Result{""}, nil, true},
		{"argument type mismatch", "addTest", []interface{}{"foo", float64(2)}, nil, fmt.Errorf("argument %d type missmatch for function %s\ngot %v\nwant %v", 1, "addTest", reflect.TypeOf("foo"), reflect.TypeOf(float64(0))), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			called = false
			got, err := en.processSliceFunc(tc.fname, tc.args...)
			if !reflect.DeepEqual(got, tc.want) || !reflect.DeepEqual(err, tc.wantErr) {
				t.Fatalf("processSliceFunc(%q, %v) = (%v, %v), want (%v, %v)", tc.fname, tc.args, got, err, tc.want, tc.wantErr)
			}
			if !reflect.DeepEqual(called, tc.wantCalled) {
				t.Errorf("function called = %v, want %v", called, tc.wantCalled)
			}
		})
	}
}

func TestProcess(t *testing.T) {
	tests := []struct {
		name        string
		input       interface{}
		nested      bool
		want        interface{}
		wantErr     error
		wantMessage string
	}{
		{
			name:  "root single result",
			input: []interface{}{"!not", true},
			want:  false,
		},
		{
			name:  "root no results",
			input: []interface{}{"!comment"},
		},
		{
			name:        "root multiple results",
			input:       []interface{}{"!not", true, false},
			want:        Result{false, true},
			wantErr:     ErrorMultipleValues{Values: Result{false, true}},
			wantMessage: "multiple values retured: funson.Result{false, true}",
		},
		{
			name:   "nested multiple results",
			input:  []interface{}{"!not", true, false},
			nested: true,
			want:   Result{false, true},
		},
		{
			name:   "nested no results",
			input:  []interface{}{"!comment"},
			nested: true,
			want:   Result{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			en := &EnviromentNode{Enviroment{}, nil}
			if tc.nested {
				en = en.Child(Enviroment{})
			}
			got, err := en.Process(tc.input)
			if !reflect.DeepEqual(got, tc.want) || !reflect.DeepEqual(err, tc.wantErr) {
				t.Fatalf("Process(%v) = (%v, %v), want (%v, %v)", tc.input, got, err, tc.want, tc.wantErr)
			}
			if err != nil && !reflect.DeepEqual(err.Error(), tc.wantMessage) {
				t.Errorf("error message = %q, want %q", err.Error(), tc.wantMessage)
			}
		})
	}
}

func TestMapErrors(t *testing.T) {
	cause := ErrorNoPath{Key: "name", Val: float64(1)}
	tests := []struct {
		name    string
		err     error
		want    string
		wantErr error
	}{
		{
			name:    "resolve function name",
			err:     ErrorResolveFunctionName{Err: cause},
			want:    "unable to resolve function name: " + cause.Error(),
			wantErr: cause,
		},
		{
			name: "empty function name",
			err:  ErrorInvalidFunctionName{Name: ""},
			want: "function name is empty or not string: \"\"",
		},
		{
			name: "non-string function name",
			err:  ErrorInvalidFunctionName{Name: float64(42)},
			want: "function name is empty or not string: 42",
		},
		{
			name:    "resolve map value",
			err:     ErrorResolveMapValue{Key: "item", Err: cause},
			want:    "unable to resolve value for key \"item\": " + cause.Error(),
			wantErr: cause,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.err.Error(), errors.Unwrap(tc.err)
			if !reflect.DeepEqual(got, tc.want) || !reflect.DeepEqual(err, tc.wantErr) {
				t.Fatalf("error message and cause = (%q, %v), want (%q, %v)", got, err, tc.want, tc.wantErr)
			}
		})
	}
}
