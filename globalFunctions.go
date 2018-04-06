package funson

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"
)

//used for variadic function results
type Result []interface{}

var functions map[string]interface{} = map[string]interface{}{}

//Adds custom function "fun" to be used as "name" in funson
//"fun" has to be in format:
//		func(*funson.EnviromentNode, [arguments you want]) [your results]
//or if number of results is variable, use
//		func(*funson.EnviromentNode, [arguments you want]) funson.Result
func AddFun(name string, fun interface{}) error {
	//fmt.Println("addFunc", name)
	//defer fmt.Println("addFunc", name, "out")
	if fun == nil {
		return fmt.Errorf("function \"fun\" is nil")
	}

	if _, ok := functions[name]; ok {
		return fmt.Errorf("duplicate function name: %s", name)
	}

	t := reflect.TypeOf(fun)
	if t.Kind() != reflect.Func {
		return fmt.Errorf("not a function: %s", t)
	}
	if t.NumIn() == 0 || t.In(0) != reflect.TypeOf(&EnviromentNode{}) {
		return fmt.Errorf("function has to have at least 1 argument and first argument has to be *EnviromentNode type: %#v", fun)
	}
	functions[name] = fun
	return nil
}

func init() {
	AddFun("not", func(_ *EnviromentNode, a bool, va ...bool) (bool, Result) {
		res := make(Result, len(va))
		for i, b := range va {
			res[i] = !b
		}
		return !a, res
	})
	AddFun("and", func(en *EnviromentNode, a bool, va ...interface{}) bool {
		if a == false {
			return false
		}
		for _, i := range va {
			ri, err := en.Process(i)
			if err != nil {
				panic(fmt.Sprintf("and: process %v failed: %s", i, err))
			}
			rr, ok := ri.(Result)
			if !ok {
				rr = Result{rr}
			}
			for _, bi := range rr {
				b, bok := bi.(bool)
				if !bok {
					panic(fmt.Sprintf("and: process result %v not bool: %s, %#v", i, reflect.TypeOf(bi), bi))
				}
				if b == false {
					return false
				}
			}
		}
		return true
	})
	AddFun("or", func(en *EnviromentNode, a bool, va ...interface{}) bool {
		if a == true {
			return true
		}
		for _, i := range va {
			ri, err := en.Process(i)
			if err != nil {
				panic(fmt.Sprintf("or: process %v failed: %s", i, err))
			}
			rr, ok := ri.(Result)
			if !ok {
				rr = Result{rr}
			}
			for _, bi := range rr {
				b, bok := bi.(bool)
				if !bok {
					panic(fmt.Sprintf("or: process result %v not bool: %s, %#v", i, reflect.TypeOf(bi), bi))
				}
				if b == true {
					return true
				}
			}
		}
		return false
	})
	AddFun("eq", func(en *EnviromentNode, a, b interface{}) bool {
		//log.Printf("\neq:")
		//log.Printf("eq: a, b: %#v, %#v", a, b)
		var err error
		a, err = en.Process(a)
		if err != nil {
			panic(fmt.Sprintf("eq: processing a %v failed: %s", a, err))
		}
		if _, ok := a.(Result); !ok {
			a = Result{a}
		}
		b, err = en.Process(b)
		if err != nil {
			panic(fmt.Sprintf("eq: processing b %v failed: %s", b, err))
		}
		if _, ok := b.(Result); !ok {
			b = Result{b}
		}
		//log.Printf("eq: processed a, b: %#v, %#v", a, b)
		return reflect.DeepEqual(a, b)
	})
	AddFun("for", func(en *EnviromentNode, cond interface{}, funs ...interface{}) Result {
		//log.Printf("\nfor:")
		res := Result{}
		if !isSliceFunc(cond) {
			panic(fmt.Sprintf("for: condition has to be a function: %v", cond))
		}
		for k := 0; true; k++ {
			ne := en.Child(Enviroment{
				"\\": map[string]interface{}{
					"i": float64(k),
				},
			})
			//log.Printf("for[%d]: cond: %v", k, cond)
			if !isSliceFunc(cond) {
				panic(fmt.Sprintf("for: [%d]: condition has to be a function: %v", k, cond))
			}
			cres, err := ne.Process(cond)
			if err != nil {
				panic(fmt.Sprintf("for: [%d]: condition process %v failed: %s", k, cond, err))
			}
			//log.Printf("for[%d]: cond result: %#v", k, cres)
			if crr, ok := cres.(Result); ok {
				if len(crr) != 1 {
					panic(fmt.Sprintf("for: [%d]: condition's variadic result has to have only one return value: %v", k, crr))
				}
				cres = crr[0]
			}
			var cb, ok bool
			if cb, ok = cres.(bool); !ok {
				panic(fmt.Sprintf("for: [%d]: condition's result has to be boolean: %s, %#v", k, reflect.TypeOf(cres), cres))
			}
			if cb == false {
				break
			}

			for f, i := range funs {
				nfe := ne.Child(Enviroment{
					".": res,
				})
				pres, err := nfe.Process(i)
				if err != nil {
					panic(fmt.Sprintf("for: [%d]: function %d process %v failed: %s", k, f, i, err))
				}
				if prr, ok := pres.(Result); ok {
					res = append(res, prr...)
					continue
				}
				res = append(res, pres)
			}
		}
		return res
	})
	AddFun("time.Format", func(en *EnviromentNode, t time.Time, l string) string {
		return t.Format(l)
	})
	AddFun("time.Now", func(en *EnviromentNode) time.Time {
		return time.Now()
	})
	AddFun("add", func(en *EnviromentNode, a, b float64) float64 {
		return a + b
	})
	AddFun("sub", func(en *EnviromentNode, a, b float64) float64 {
		return a - b
	})
	AddFun("mul", func(en *EnviromentNode, a, b float64) float64 {
		return a * b
	})
	AddFun("ceil", func(en *EnviromentNode, a float64) float64 {
		return math.Ceil(a)
	})
	AddFun("floor", func(en *EnviromentNode, a float64) float64 {
		return math.Ceil(a)
	})
	AddFun("round", func(_ *EnviromentNode, f float64) float64 {
		return round(f)
	})
	AddFun("roundN", func(_ *EnviromentNode, f float64, n float64) float64 {
		in := int(n)
		if n != float64(in) {
			panic(fmt.Sprintf("roundN: n is not integer: %f", n))
		}
		if in == 0 {
			return round(f)
		}
		absn := in
		if absn < 0 {
			absn *= -1
		}
		exp := 1.0
		for i := 0; i < absn; i++ {
			exp *= 10
		}
		if in < 0 {
			exp = 1 / exp
		}
		return round(f*exp) / exp
	})
	AddFun("div", func(en *EnviromentNode, a, b float64) float64 {
		if b == 0 {
			panic(fmt.Sprintf("division by 0"))
		}
		return a / b
	})
	AddFun("sum", func(en *EnviromentNode, nums ...float64) float64 {
		res := float64(0)
		for _, n := range nums {
			res += n
		}
		return res
	})
	AddFun("env", func(en *EnviromentNode, path string) interface{} {
		//fmt.Printf("\nenv: p: %#v\n", path)
		//fmt.Printf("env: e: %v\n", e)
		//defer fmt.Printf("env: end\n\n")
		path = strings.TrimSpace(path)

		switch path[0] {
		case '.', ':', '\\':
			dot, ok := en.FirstKey(string([]byte{path[0]}))
			if !ok {
				panic(fmt.Sprintf("env: no \"%s\" in enviroments: %v", string(path[0]), path))
			}
			res, err := pathFunc(dot, path[1:])
			if err != nil {
				panic(fmt.Sprintf("cannot resolve path \"%s\" in %#v: %s", path[1:], dot, err))
			}
			return res
		default:
			panic(fmt.Sprintf("env: unknown path prefix \"%v\"", string(path[0])))
		}
		panic(fmt.Sprintf("out of switch: %v", path))
		return nil
	})
	AddFun("pairsToMap", func(en *EnviromentNode, pairs ...interface{}) map[string]interface{} {
		out := map[string]interface{}{}
		for i, pairUntyped := range pairs {
			pair, ok := pairUntyped.([]interface{})
			if !ok || len(pair) != 2 {
				panic(fmt.Sprintf("pairsToMap: item #%d is not valid pair: pair has to be slice of length 2, not: %#v", i, pair))
			}
			key, ok := pair[0].(string)
			if !ok {
				panic(fmt.Sprintf("pairsToMap: item #%d is not valid pair: first item has to be string, not: %#v", i, pair[0]))
			}
			if _, ok := out[key]; ok {
				panic(fmt.Sprintf("pairsToMap: item #%d is not valid pair: duplicate pair key: %s", i, key))
			}
			en.Enviroment[":"] = out
			val, err := en.Process(pair[1])
			if err != nil {
				panic(fmt.Sprintf("pairsToMap: can not compute value for key %s : %s", key, err))
			}
			if res, ok := val.(Result); ok {
				switch len(res) {
				case 0:
					out[key] = nil
				case 1:
					out[key] = res[0]
				default:
					panic(fmt.Sprintf("pairsToMap: to many results: %#v", res))
				}
				continue
			}
			out[key] = val
		}
		return out
	})
}

func pathFunc(i interface{}, path string) (interface{}, error) {
	if path == "" {
		return i, nil
	}
	ps := strings.SplitN(path, ".", 2)
	if len(ps) == 1 {
		ps = append(ps, "")
	}
	head, npath := ps[0], ps[1]
	switch it := i.(type) {
	case map[string]interface{}:
		ni, ok := it[head]
		if !ok {
			return nil, fmt.Errorf("no \"%s\" in map[string]interface{}: %v", head, it)
		}
		return pathFunc(ni, npath)
	case []interface{}:
		res := Result{}
		for _, ni := range it {
			ri, err := pathFunc(ni, path)
			if err != nil {
				return nil, err
			}
			if r, ok := ri.(Result); ok {
				res = append(res, r...)
				continue
			}
			res = append(res, ri)
		}
		return res, nil
	}
	return nil, fmt.Errorf("no \"%s\" in %#v", head, i)
}
func round(f float64) float64 {
	if f <= -0.5 {
		return float64(int(f - 0.5))
	}
	if f >= 0.5 {
		return float64(int(f + 0.5))
	}
	return 0
}
