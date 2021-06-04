package funson

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mohae/deepcopy"
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
	AddFun("comment", func(en *EnviromentNode, _ ...interface{}) interface{} {
		return Result{}
	})
	AddFun("if", func(en *EnviromentNode, cond bool, resTrue, resFalse interface{}) interface{} {
		//log.Printf("if(cond: %#v, resTrue: %#v, resFalse: %#v)", cond, resTrue, resFalse)
		unporcessedResut := resFalse
		if cond {
			unporcessedResut = resTrue
		}

		res, err := en.Process(unporcessedResut)
		if err != nil {
			panic(err.Error())
		}
		return res
	})
	AddFun("not", func(_ *EnviromentNode, a bool, va ...bool) (bool, Result) {
		res := make(Result, len(va))
		for i, b := range va {
			res[i] = !b
		}
		return !a, res
	})
	AddFun("?and", func(en *EnviromentNode, a bool, va ...interface{}) bool {
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
	AddFun("?or", func(en *EnviromentNode, a bool, va ...interface{}) bool {
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
	AddFun("?eq", func(en *EnviromentNode, a, b interface{}) bool {
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
	AddFun("?env", func(en *EnviromentNode, path string) bool {
		//log.Printf("isEnv(path: %#v)", path)
		path = strings.TrimSpace(path)

		switch path[0] {
		case '.', ':', '\\':
			dot, ok := en.FirstKey(string([]byte{path[0]}))
			if !ok {
				return false
			}
			return isPathFunc(dot, path[1:])
		default:
			panic(fmt.Sprintf("isEnv: unknown path prefix \"%v\"", string(path[0])))
		}
		panic(fmt.Sprintf("isEnv: out of switch: %v", path))
		return false
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
	AddFun("item", func(en *EnviromentNode, index float64, array []interface{}) interface{} {
		//log.Printf("item: params: {index: %f, array: %#v}", index, array)
		n := int(index)
		if float64(n) != index {
			panic(fmt.Sprintf("item: index is not integer: %f", index))
		}

		processed, err := en.Process(array)
		if err != nil {
			panic(err.Error())
		}
		//log.Printf("item: pocessed array: %#v", processed)

		if processedResult, ok := processed.(Result); ok {
			//log.Printf("item: pocessed array is Result: %#v", processedResult)
			if len(processedResult) != 1 {
				panic(fmt.Sprintf("item: processed array needs only 1 output, got %d: %v", len(processedResult), processedResult))
			}

			processedResultValue := reflect.ValueOf(processedResult[0])
			processedResultKind := processedResultValue.Kind()
			if processedResultKind != reflect.Array && processedResultKind != reflect.Slice {
				panic(fmt.Sprintf("item: processed array output is not array, nor slice, got: %s", processedResultKind))
			}

			res := processedResultValue.Index(n).Interface()
			//log.Printf("item: returning %#v", res)
			return res
		} else {
			//TODO test this
			if slice, ok := processed.([]interface{}); !ok {
				panic(fmt.Sprintf("item: processed array is not []interface{}: %T", processed))
			} else {
				array = slice
			}
		}
		//log.Printf("item: pocessed array result: %#v", array)

		return array[n]
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
	AddFun("concat", func(_ *EnviromentNode, what ...string) string {
		//log.Printf("concat(what: %#v)", what)
		return strings.Join(what, "")
	})
	AddFun("split", func(_ *EnviromentNode, byWhat, where string) []string {
		return strings.Split(where, byWhat)
	})
	AddFun("replacePrefix", func(en *EnviromentNode, find, replace, where string) string {
		//log.Printf("replacePrefix(find: %#v, replace: %#v, where: %#v)", find, replace, where)
		if !strings.HasPrefix(where, find) {
			return where
		}

		return replace + strings.TrimPrefix(where, find)
	})
	AddFun("input", input)
	AddFun("choose", choose)
}

func isPathFunc(i interface{}, path string) bool {
	if path == "" {
		return true
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
			return false
		}
		return isPathFunc(ni, npath)
	case []interface{}:
		for _, ni := range it {
			if isPathFunc(ni, path) {
				return true
			}
		}
		return false
	}
	return false
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

type timeFormat struct {
	input, output string
}

func stringRetype(_type, input string, datetimeFormat timeFormat) (interface{}, error) {
	switch _type {
	case "string":
		return input, nil
	case "float":
		inputFloat, err := strconv.ParseFloat(input, 64)
		if err != nil {
			return nil, fmt.Errorf("\"%s\" is not a number", input)
		}
		return inputFloat, nil
	case "integer":
		inputInt, err := strconv.Atoi(input)
		if err != nil {
			return nil, fmt.Errorf("\"%s\" is not an integer", input)
		}
		return inputInt, nil
	case "datetime":
		t, err := time.Parse(datetimeFormat.input, input)
		if err != nil {
			return nil, fmt.Errorf("\"%s\" is not a date in format \"%s\"", input, datetimeFormat.input)
		}
		return t.Format(datetimeFormat.output), nil
	}
	panic(fmt.Sprintf("input: can not retype input string to: %s", _type))
	return nil, nil
}

var reader *bufio.Reader = bufio.NewReader(os.Stdin)

func input(en *EnviromentNode, o map[string]interface{}) interface{} {
	if _, ok := o["type"]; !ok {
		o["type"] = "string"
	} else if _, ok := o["type"].(string); !ok {
		panic(fmt.Sprintf("input: want \"string\" type of \"type\": %v", reflect.TypeOf(o["type"])))
	}
	_type := o["type"].(string)

	switch _type {
	case "string", "float", "integer", "datetime":
	default:
		panic(fmt.Sprintf("input: unknown \"type\": %s", _type))
	}

	_datetimeFormat := timeFormat{}
	if _type == "datetime" {
		if _, ok := o["datetime-format-input"]; !ok {
			o["datetime-format-input"] = time.RFC822
		} else if _, ok := o["datetime-format-input"].(string); !ok {
			panic(fmt.Sprintf("input: want \"string\" type of \"datetime-format-input\": %v", reflect.TypeOf(o["datetime-format-input"])))
		}
		_datetimeFormat.input = o["datetime-format-input"].(string)

		if _, ok := o["datetime-format-output"]; !ok {
			o["datetime-format-output"] = time.RFC822
		} else if _, ok := o["datetime-format-output"].(string); !ok {
			panic(fmt.Sprintf("input: want \"string\" type of \"datetime-format-output\": %v", reflect.TypeOf(o["datetime-format-output"])))
		}
		_datetimeFormat.output = o["datetime-format-output"].(string)
	}

	if _, ok := o["question"]; !ok {
		o["question"] = "Enter input"
	} else if _, ok := o["question"].(string); !ok {
		panic(fmt.Sprintf("input: want \"string\" type for \"question\": %v", reflect.TypeOf(o["question"])))
	}
	_question := o["question"].(string)

	//TODO do not change predefined
	if pi, ok := o["predefined"]; ok {
		nen := en.Child(Enviroment{
			":": o,
		})
		npi, err := nen.Process(pi)
		if err != nil {
			panic(fmt.Sprintf("input: processing \"predefined\" error: %s", err))
		}
		if npr, ok := npi.(Result); ok {
			if len(npr) != 1 {
				panic(fmt.Sprintf("input: want only 1 \"predefined\"  result: %v", npr))
			}
			npi = npr[0]
		}
		o["predefined"] = npi
	}

	if _, ok := o["predefined"]; !ok {
		o["predefined"] = ""
	} else if f, ok := o["predefined"].(float64); ok {
		o["predefined"] = fmt.Sprintf("%f", f)
	} else if _, ok := o["predefined"].(string); !ok {
		panic(fmt.Sprintf("input: want \"string\" type for \"predefined\": %v", reflect.TypeOf(o["predefined"])))
	}
	_predefined := o["predefined"].(string)
	if _predefined != "" {
		_question += " [" + _predefined + "]"
	}

	var _validator *regexp.Regexp
	if _, ok := o["validator"]; !ok {
		o["validator"] = ""
	} else if _, ok := o["validator"].(string); !ok {
		panic(fmt.Sprintf("input: want \"string\" type for \"validator\": %v", reflect.TypeOf(o["validator"])))
	}
	_expression := o["validator"].(string)
	if _expression != "" {
		var err error
		_validator, err = regexp.Compile(_expression)
		if err != nil {
			panic(fmt.Sprintf("input: \"validator\" is not regexp compilant: %s", err))
		}
	}

	if _validator != nil && _predefined != "" {
		if !_validator.Match([]byte(_predefined)) {
			panic(fmt.Sprintf("input: \"predefined\" doesn't pass \"validator\""))
		}
	}

	if _predefined != "" {
		if _, err := stringRetype(_type, _predefined, _datetimeFormat); err != nil {
			panic(fmt.Sprintf("String \"predefined\" %s", err))
		}
	}

	if _, ok := o["condition"]; !ok {
		o["condition"] = ""
	} else if _, ok := o["condition"].(string); !ok {
		panic(fmt.Sprintf("input: want \"string\" type for \"condition\": %v", reflect.TypeOf(o["condition"])))
	}
	_condition := o["condition"].(string)

	if _validator != nil && _condition == "" {
		panic(fmt.Sprintf("input: you forgot to fill \"validator\" description into \"condition\" field"))
	}

	var res interface{}

	for res == nil {
		fmt.Printf("\n%s: ", _question)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" && _predefined != "" {
			input = _predefined
		}

		if _validator != nil {
			if !_validator.Match([]byte(input)) {
				fmt.Println("Entered value doesn't pass condition.")
				fmt.Println(_condition)
				continue
			}
		}
		inputRetyped, err := stringRetype(_type, input, _datetimeFormat)
		if err != nil {
			fmt.Printf("Entered value %s", err)
			continue
		}
		res = inputRetyped
	}
	return res
}

func choose(en *EnviromentNode, o map[string]interface{}) interface{} {
	//log.Printf("\nchoose: %v", o)
	//defer log.Printf("choose: end")
	if _, ok := o["options"].([]interface{}); !ok {
		panic(fmt.Sprintf("input: want \"[]interface{}\" type for \"options\": %v", reflect.TypeOf(o["options"])))
	}
	_options := o["options"].([]interface{})

	if len(_options) == 0 {
		if o["predefined"] != nil {
			return o["predefined"]
		}
		return Result{}
	}

	options := make([]struct {
		text   string
		option interface{}
	}, len(_options))

	for i, _o := range _options {
		switch _oTyped := _o.(type) {
		case []interface{}:
			if len(_oTyped) != 2 {
				panic(fmt.Sprintf("choose: %d option is slice and has to have only two items (option, text): %v", i, _o))
			}
			options[i].option = _oTyped[1]
			ne := en.Child(Enviroment{
				":": _oTyped[1],
			})
			ores, err := ne.Process(_oTyped[0])
			if err != nil {
				panic(fmt.Sprintf("choose: processing option %d text failed: %s", i, err))
			}
			if orr, ok := ores.(Result); ok {
				if len(orr) != 1 {
					panic(fmt.Sprintf("choose: want only one option %d text result: %v", i, orr))
				}
				ores = orr[0]
			}
			options[i].text = fmt.Sprintf("%v", ores)
		default:
			//TODO duplicate code
			options[i].option = _o
			options[i].text = fmt.Sprintf("%v", _o)
			if ot, ok := o["option-text"]; ok {
				ne := en.Child(Enviroment{
					":": _o,
				})
				_ores, err := ne.Process(ot)
				if err != nil {
					panic(fmt.Sprintf("choose: processing option %d option-text failed: %s", i, err))
				}
				if orr, ok := _ores.(Result); ok {
					if len(orr) != 1 {
						panic(fmt.Sprintf("choose: want only one option %d text result: %v", i, orr))
					}
					_ores = orr[0]
				}
				options[i].text = fmt.Sprintf("%v", _ores)
			}
		}
	}

	if _, ok := o["question"]; !ok {
		o["question"] = "Choose an option"
		if o["predefined"] != nil {
			o["question"] = o["question"].(string) + " or don't"
		}
	} else if _, ok := o["question"].(string); !ok {
		panic(fmt.Sprintf("input: want \"string\" type for \"question\": %v", reflect.TypeOf(o["question"])))
	}
	_question := o["question"].(string)

	var res interface{}
	for res == nil {
		fmt.Printf("\nOptions:\n")
		for i, option := range options {
			fmt.Printf("%d) %s\n", i+1, option.text)
		}
		fmt.Printf("%s: ", _question)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			if o["predefined"] != nil {
				defres, err := en.Process(o["predefined"])
				if err != nil {
					panic(fmt.Sprintf("choose: processing predefined result failed: %s", err))
				}
				return defres
			}
			fmt.Printf("You have to choose some option\n")
			continue
		}
		inputInt, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("Choose by entering option number")
			continue
		}
		if inputInt < 1 || inputInt > len(options) {
			fmt.Println("Choose a number from list.")
			continue
		}
		res = options[inputInt-1].option
	}
	res, err := en.Process(res)
	if err != nil {
		panic(fmt.Sprintf("choose: processing choosen result failed: %s", err))
	}
	if op, ok := o["option-process"]; ok {
		ne := en.Child(Enviroment{
			"\\": res,
		})
		//log.Printf("choose: %v.Process(%#v)", ne.Enviroment, op)
		//TODO dont usecopy, repair input
		opres, err := ne.Process(deepcopy.Copy(op))
		if err != nil {
			panic(fmt.Sprintf("choose: processing option-process for choosen result failed: %s", err))
		}
		res = opres
	}
	return res
}
