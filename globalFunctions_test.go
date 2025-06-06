package funson

import (
	"reflect"
	"testing"
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
