package funson

import (
	"reflect"
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
		wantErr   bool
	}{
		{
			name:      "valid function",
			functions: nil,
			funName:   "dummy",
			fun:       dummy,
			wantErr:   false,
		},
		{
			name:      "nil function",
			functions: nil,
			funName:   "nilfun",
			fun:       nil,
			wantErr:   true,
		},
		{
			name:      "duplicate name",
			functions: map[string]interface{}{"dup": dummy},
			funName:   "dup",
			fun:       other,
			wantErr:   true,
		},
		{
			name:      "non-function value",
			functions: nil,
			funName:   "notfun",
			fun:       123,
			wantErr:   true,
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
			if (err != nil) != tc.wantErr {
				t.Fatalf("AddFun(%q) error = %v, wantErr %v", tc.funName, err, tc.wantErr)
			}

			if tc.wantErr {
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
