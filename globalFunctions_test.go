package funson

import (
	"bufio"
	"reflect"
	"strings"
	"testing"
	"time"
)

// copyMap returns a shallow copy of the provided map.
func copyMap(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func TestAddFun(t *testing.T) {
	origFunctions := copyMap(functions)
	defer func() { functions = origFunctions }()

	dummy := func(en *EnviromentNode) {}
	other := func(en *EnviromentNode) {}

	tests := []struct {
		name      string
		functions map[string]interface{}
		funName   string
		fun       interface{}
		wantErr   error
	}{
		{
			name:      "valid function",
			functions: nil,
			funName:   "dummy",
			fun:       dummy,
			wantErr:   nil,
		},
		{
			name:      "nil function",
			functions: nil,
			funName:   "nilfun",
			fun:       nil,
			wantErr:   ErrorNilFunction{},
		},
		{
			name:      "duplicate name",
			functions: map[string]interface{}{"dup": dummy},
			funName:   "dup",
			fun:       other,
			wantErr:   ErrorDuplicateFunctionName{Name: "dup"},
		},
		{
			name:      "non-function value",
			functions: nil,
			funName:   "notfun",
			fun:       123,
			wantErr:   ErrorNotFunction{Type: reflect.TypeOf(123)},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.functions == nil {
				functions = map[string]interface{}{}
			} else {
				functions = copyMap(tc.functions)
			}
			original := copyMap(functions)

			err := AddFun(tc.funName, tc.fun)
			if !reflect.DeepEqual(err, tc.wantErr) {
				t.Fatalf("AddFun(%q) error = %v, wantErr %v", tc.funName, err, tc.wantErr)
			}
			if tc.wantErr != nil {
				if len(functions) != len(original) {
					t.Errorf("functions map size changed on error: got %d want %d", len(functions), len(original))
				}
				for k, v := range original {
					got, ok := functions[k]
					if !ok {
						t.Errorf("missing function %q after error", k)
						continue
					}
					if reflect.ValueOf(got).Pointer() != reflect.ValueOf(v).Pointer() {
						t.Errorf("function %q changed", k)
					}
				}
				return
			}

			if len(functions) != len(original)+1 {
				t.Errorf("len(functions) = %d, want %d", len(functions), len(original)+1)
			}
			for k, v := range original {
				if functions[k] != v {
					t.Errorf("existing function %q changed", k)
				}
			}
			got, ok := functions[tc.funName]
			if !ok {
				t.Errorf("function %q not registered", tc.funName)
			} else if reflect.ValueOf(got).Pointer() != reflect.ValueOf(tc.fun).Pointer() {
				t.Errorf("registered function mismatch")
			}
		})
	}
}

func TestPathFuncAndIsPathFunc(t *testing.T) {
	tests := []struct {
		name    string
		data    interface{}
		path    string
		want    interface{}
		wantErr error
	}{
		{
			name:    "map path",
			data:    map[string]interface{}{"a": map[string]interface{}{"b": float64(42)}},
			path:    "a.b",
			want:    float64(42),
			wantErr: nil,
		},
		{
			name: "slice root",
			data: []interface{}{
				map[string]interface{}{"foo": float64(1)},
				map[string]interface{}{"foo": float64(2)},
			},
			path:    "foo",
			want:    Result{float64(1), float64(2)},
			wantErr: nil,
		},
		{
			name: "nested slice",
			data: map[string]interface{}{
				"list": []interface{}{
					map[string]interface{}{"val": float64(5)},
					map[string]interface{}{"val": float64(8)},
				},
			},
			path:    "list.val",
			want:    Result{float64(5), float64(8)},
			wantErr: nil,
		},
		{
			name:    "missing key",
			data:    map[string]interface{}{"a": float64(1)},
			path:    "b",
			want:    nil,
			wantErr: ErrorMapKeyMissing{Key: "b", Map: map[string]interface{}{"a": float64(1)}},
		},
		{
			name:    "nested missing",
			data:    map[string]interface{}{"a": map[string]interface{}{"b": float64(1)}},
			path:    "a.c",
			want:    nil,
			wantErr: ErrorMapKeyMissing{Key: "c", Map: map[string]interface{}{"b": float64(1)}},
		},
		{
			name: "slice missing",
			data: []interface{}{
				map[string]interface{}{"a": float64(1)},
			},
			path:    "b",
			want:    nil,
			wantErr: ErrorMapKeyMissing{Key: "b", Map: map[string]interface{}{"a": float64(1)}},
		},
		{
			name:    "invalid type",
			data:    float64(1),
			path:    "foo",
			want:    nil,
			wantErr: ErrorNoPath{Key: "foo", Val: float64(1)},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := pathFunc(tc.data, tc.path)
			if !reflect.DeepEqual(got, tc.want) || !reflect.DeepEqual(err, tc.wantErr) {
				t.Fatalf("pathFunc(%v, %q) = (%v, %v), want (%v, %v)", tc.data, tc.path, got, err, tc.want, tc.wantErr)
			}
			is := isPathFunc(tc.data, tc.path)
			if is != (tc.wantErr == nil) {
				t.Errorf("isPathFunc(%v, %q) = %v, want %v", tc.data, tc.path, is, tc.wantErr == nil)
			}
		})
	}
}

func TestRound(t *testing.T) {
	tests := []struct {
		name  string
		input float64
		want  float64
	}{
		{"zero", 0, 0},
		{"positive below half", 0.4, 0},
		{"positive half", 0.5, 1},
		{"positive just below half", 0.49, 0},
		{"positive just above half", 0.51, 1},
		{"positive above half", 1.5, 2},
		{"positive below half integer", 1.4, 1},
		{"negative above minus half", -0.4, 0},
		{"negative half", -0.5, -1},
		{"negative just above half", -0.51, -1},
		{"negative just below half", -0.49, 0},
		{"negative below half", -1.5, -2},
		{"negative above half integer", -1.4, -1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := round(tc.input)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("round(%v) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestRoundN(t *testing.T) {
	rn, ok := functions["roundN"].(func(*EnviromentNode, float64, float64) float64)
	if !ok {
		t.Fatalf("roundN has unexpected type")
	}
	en := &EnviromentNode{Enviroment{}, nil}

	tests := []struct {
		name string
		f    float64
		n    float64
		want float64
	}{
		{"n zero", 1.5, 0, 2},
		{"n positive", 1.2345, 2, 1.23},
		{"n positive round", 1.235, 2, 1.24},
		{"n negative", 15.9, -1, 20},
		{"n negative round", 149.9, -2, 100},
		{"n positive negative value", -1.234, 2, -1.23},
		{"n negative negative value", -45.6, -1, -50},
		{"n positive three decimals", 1.23456, 3, 1.235},
		{"n negative three digits", 12345.6, -3, 12000},
		{"n positive three decimals negative", -1.23456, 3, -1.235},
		{"n negative three digits negative", -12345.6, -3, -12000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := rn(en, tc.f, tc.n)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("functions[\"roundN\"](%v, %v) = %v, want %v", tc.f, tc.n, got, tc.want)
			}
		})
	}
}

func TestItem(t *testing.T) {
	it, ok := functions["item"].(func(*EnviromentNode, float64, []interface{}) interface{})
	if !ok {
		t.Fatalf("item has unexpected type")
	}
	en := &EnviromentNode{Enviroment{}, nil}

	tests := []struct {
		name      string
		index     float64
		array     []interface{}
		useChild  bool
		want      interface{}
		wantPanic bool
	}{
		{
			name:  "plain slice",
			index: 1,
			array: []interface{}{"a", "b", "c"},
			want:  "b",
		},
		{
			name:     "result slice",
			index:    1,
			array:    []interface{}{"!split", ",", "a,b,c"},
			useChild: true,
			want:     "b",
		},
		{
			name:      "non integer index",
			index:     1.5,
			array:     []interface{}{"a", "b"},
			wantPanic: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			itemEnv := en
			if tc.useChild {
				itemEnv = en.Child(Enviroment{})
			}
			if tc.wantPanic {
				defer func() {
					if recover() == nil {
						t.Errorf("item(%v, %v) did not panic", tc.index, tc.array)
					}
				}()
				it(itemEnv, tc.index, tc.array)
				return
			}
			got := it(itemEnv, tc.index, tc.array)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("item(%v, %v) = %#v, want %#v", tc.index, tc.array, got, tc.want)
			}
		})
	}
}

func TestStringRetype(t *testing.T) {
	fixedTime := time.Date(2023, time.September, 18, 7, 45, 0, 0, time.UTC)
	rfcTime := fixedTime.Format(time.RFC822)

	tests := []struct {
		name    string
		typ     string
		input   string
		tf      timeFormat
		want    interface{}
		wantErr error
	}{
		{"string", "string", "hello", timeFormat{}, "hello", nil},
		{"float ok", "float", "3.14", timeFormat{}, float64(3.14), nil},
		{"float bad", "float", "foo", timeFormat{}, nil, ErrorNotNumber{Input: "foo"}},
		{"integer ok", "integer", "42", timeFormat{}, 42, nil},
		{"integer bad", "integer", "a", timeFormat{}, nil, ErrorNotInteger{Input: "a"}},
		{"datetime default", "datetime", rfcTime, timeFormat{input: time.RFC822, output: time.RFC822}, rfcTime, nil},
		{"datetime formats", "datetime", "31.12.2021 23:59", timeFormat{input: "02.01.2006 15:04", output: "2006-01-02/15:04"}, "2021-12-31/23:59", nil},
		{"datetime bad", "datetime", "nope", timeFormat{input: "02.01.2006", output: "2006-01-02"}, nil, ErrorInvalidDate{Input: "nope", Format: "02.01.2006"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := stringRetype(tc.typ, tc.input, tc.tf)
			if !reflect.DeepEqual(got, tc.want) || !reflect.DeepEqual(err, tc.wantErr) {
				t.Fatalf("stringRetype(%q, %q) = (%v, %v), want (%v, %v)", tc.typ, tc.input, got, err, tc.want, tc.wantErr)
			}
		})
	}
}

func TestInput(t *testing.T) {
	en := &EnviromentNode{Enviroment{}, nil}
	origReader := reader
	defer func() { reader = origReader }()

	tests := []struct {
		name        string
		readerInput string
		options     map[string]interface{}
		want        interface{}
	}{
		{
			name:        "predefined used",
			readerInput: "\n",
			options:     map[string]interface{}{"type": "integer", "predefined": "42"},
			want:        42,
		},
		{
			name:        "validator retry",
			readerInput: "xx\nabc\n",
			options:     map[string]interface{}{"type": "string", "validator": "^[a-z]{3}$", "condition": "3 letters"},
			want:        "abc",
		},
		{
			name:        "float conversion",
			readerInput: "3.14\n",
			options:     map[string]interface{}{"type": "float"},
			want:        float64(3.14),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reader = bufio.NewReader(strings.NewReader(tc.readerInput))
			got := input(en, tc.options)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("input(%v) = %#v, want %#v", tc.options, got, tc.want)
			}
		})
	}
}

func TestChoose(t *testing.T) {
	en := &EnviromentNode{Enviroment{}, nil}
	origReader := reader
	defer func() { reader = origReader }()

	tests := []struct {
		name        string
		readerInput string
		options     map[string]interface{}
		want        interface{}
	}{
		{
			name:        "choose by number",
			readerInput: "2\n",
			options:     map[string]interface{}{"options": []interface{}{"a", "b", "c"}},
			want:        "b",
		},
		{
			name:        "default on empty",
			readerInput: "\n",
			options:     map[string]interface{}{"options": []interface{}{"x", "y"}, "predefined": "y"},
			want:        "y",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reader = bufio.NewReader(strings.NewReader(tc.readerInput))
			got := choose(en, tc.options)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("choose(%v) = %#v, want %#v", tc.options, got, tc.want)
			}
		})
	}
}
