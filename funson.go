package funson

import (
	"fmt"
	"reflect"
)

type Enviroment map[string]interface{}

type EnviromentNode struct {
	Enviroment
	parent *EnviromentNode
}

func (en *EnviromentNode) Child(e Enviroment) *EnviromentNode {
	return &EnviromentNode{e, en}
}

func (en *EnviromentNode) Parent() *EnviromentNode {
	if en == nil {
		return nil
	}
	return en.parent
}

func (en *EnviromentNode) FirstKey(k string) (interface{}, bool) {
	for en != nil {
		if v, ok := en.Enviroment[k]; ok {
			return v, true
		}
		en = en.Parent()
	}
	return nil, false
}

func Fun(in interface{}) (res interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("no Fun: %s", r)
		}
	}()
	env := &EnviromentNode{
		Enviroment{},
		nil,
	}
	res, err = env.Process(in)
	return
}

func isSliceFunc(i interface{}) bool {
	s, sok := i.([]interface{})
	if !sok {
		return false
	}
	name, ok := s[0].(string)
	return ok && len(name) >= 2 && name[0] == '!' && name[1] != '!'
}

func sliceFunc(sf []interface{}) (string, []interface{}) {
	if !isSliceFunc(sf) {
		return "", nil
	}
	return sf[0].(string)[1:], sf[1:]

}

func (e *EnviromentNode) processSliceFunc(name string, args ...interface{}) (interface{}, error) {
	//log.Printf("\nproccessSliceFunc: %s: %v\n", name, args)
	//log.Printf("processSliceFunc: e: %#v\n", e.Enviroment)
	//defer log.Printf("processSliceFunc: end\n\n")

	function, ok := functions[name]
	if !ok {
		return nil, fmt.Errorf("no function found: %s", name)
	}

	t := reflect.TypeOf(function)

	inputs := make([]reflect.Value, t.NumIn())
	inputs[0] = reflect.ValueOf(e)
	fillInputsTo := t.NumIn()
	if t.IsVariadic() {
		fillInputsTo--
	}
	processedArgs := make([]interface{}, 0)
	for i := 1; i < fillInputsTo; i++ {
		var inArg interface{}
		isProcessed := false
		if len(processedArgs) > 0 {
			inArg = processedArgs[0]
			processedArgs = processedArgs[1:]
			isProcessed = true

		} else if len(args) > 0 {
			inArg = args[0]
			args = args[1:]

		} else {
			return nil, fmt.Errorf("not enough arguments for function %s: %v", name, inputs)
		}
		it := t.In(i)
		if it == reflect.TypeOf(inArg) || reflect.TypeOf(inArg).ConvertibleTo(it) {
			inputs[i] = reflect.ValueOf(inArg)
			continue
		}
		if isProcessed {
			return nil, fmt.Errorf("argument %d type missmatch for function %s\ngot %v\nwant %v", i, name, reflect.TypeOf(inArg), it)
		}
		ri, err := e.Process(inArg)
		if err != nil {
			return nil, fmt.Errorf("error processing %d argument for function %s: %s", i, name, err)
		}
		if r, ok := ri.(Result); ok {
			if len(r) == 0 {
				i--
				continue
			}
			processedArgs = append(processedArgs, r[1:]...)
			ri = r[0]
		}
		if it == reflect.TypeOf(ri) || reflect.TypeOf(ri).ConvertibleTo(it) {
			inputs[i] = reflect.ValueOf(ri)
			continue
		}
		return nil, fmt.Errorf("argument %d type missmatch for function %s\ngot %v\nwant %v", i, name, reflect.TypeOf(ri), it)
	}

	if t.IsVariadic() {
		i := len(inputs) - 1
		vat := t.In(i)
		vaet := vat.Elem()
		inv := reflect.MakeSlice(vat, 0, len(processedArgs)+len(args))
		for len(processedArgs)+len(args) > 0 {
			var inArg interface{}
			isProcessed := false
			if len(processedArgs) > 0 {
				inArg = processedArgs[0]
				processedArgs = processedArgs[1:]
				isProcessed = true
			} else if len(args) > 0 {
				inArg = args[0]
				args = args[1:]
			} else {
				break
			}
			if vaet == reflect.TypeOf(inArg) || reflect.TypeOf(inArg).ConvertibleTo(vaet) {
				inv = reflect.Append(inv, reflect.ValueOf(inArg))
				continue
			}
			if isProcessed {
				return nil, fmt.Errorf("variadic argument type missmatch for function %s\ngot %v\nwant %v", name, reflect.TypeOf(inArg), vaet)
			}

			ri, err := e.Process(inArg)
			if err != nil {
				return nil, fmt.Errorf("error processing variadic argument for function %s: %s", name, err)
			}
			if r, ok := ri.(Result); ok {
				if len(r) == 0 {
					continue
				}
				processedArgs = append(processedArgs, r[1:]...)
				ri = r[0]
			}
			if vaet == reflect.TypeOf(ri) || reflect.TypeOf(ri).ConvertibleTo(vaet) {
				inv = reflect.Append(inv, reflect.ValueOf(ri))
				continue
			}
			return nil, fmt.Errorf("variadic argument type missmatch for function %s\ngot %v\nwant %v", name, reflect.TypeOf(ri), vaet)
		}
		inputs[i] = inv
	}
	//TODO If function is not variadic, check if here are any leftover arguments and return error if any.

	var resVal []reflect.Value
	if t.IsVariadic() {
		resVal = reflect.ValueOf(function).CallSlice(inputs)
	} else {
		resVal = reflect.ValueOf(function).Call(inputs)
	}
	res := make(Result, 0, len(resVal))
	for i := range resVal {
		ri := resVal[i].Interface()
		if rr, ok := ri.(Result); ok {
			res = append(res, rr...)
			continue
		}
		res = append(res, ri)
	}
	return res, nil
}

func (e *EnviromentNode) processSlice(in []interface{}) (interface{}, error) {
	//log.Printf("\nproccessSlice: i: %v\n", in)
	//log.Printf("processSlice: e: %#v\n", e.Enviroment)
	//defer log.Printf("processSlice: end\n\n")

	if len(in) == 0 {
		return []interface{}{}, nil
	}

	out := make([]interface{}, 0, len(in))
	for i, v := range in {
		//log.Printf("\tout: %#v\n", out)
		//log.Printf("\tprocessing %d item %v\n", i, v)

		e.Enviroment["."] = out
		ri, err := e.Process(v)
		if err != nil {
			return out, fmt.Errorf("processing %d item error: %s", i, err)
		}
		//log.Printf("\tcomputing result: %#v\n", ri)
		if r, ok := ri.(Result); ok {
			out = append(out, r...)
			continue
		}
		out = append(out, ri)
	}
	//log.Printf("out: %#v\n", out)
	return out, nil
}

func (e *EnviromentNode) Process(in interface{}) (interface{}, error) {
	switch typedIn := in.(type) {
	case []interface{}:
		var res interface{}
		var err error

		if isSliceFunc(typedIn) {
			name, args := sliceFunc(typedIn)
			res, err = e.Child(Enviroment{
				"type": "sliceFunc",
				"name": name,
			}).processSliceFunc(name, args...)
		} else {
			res, err = e.Child(Enviroment{
				"type": "slice",
			}).processSlice(typedIn)
		}
		if err != nil {
			return res, err
		}
		fr, ok := res.(Result)
		if !ok {
			return res, nil
		}
		if e.Parent() == nil {
			if len(fr) == 0 {
				return nil, nil
			}
			if len(fr) > 1 {
				//TODO structured error
				return fr, fmt.Errorf("multiple values retured: %#v", fr)
			}
			return fr[0], nil
		}
		return fr, nil
	case map[string]interface{}:
		//return runMap(e.Copy(), typedIn)
	case string:
		if t, ok := e.Enviroment["type"].(string); ok && t == "slice" {
			if o, ok := e.Enviroment[":"].([]interface{}); ok && len(o) == 0 {
				if len(typedIn) >= 2 && typedIn[0] == '!' && typedIn[1] == '!' {
					//log.Printf("process: first string in array begins with \"!!\": %s", typedIn)
					return typedIn[1:], nil
				}
			}
		}
	}
	//log.Printf("process: type: %s", reflect.TypeOf(in))
	return in, nil
}

//func runMap(env Enviroment, in map[string]interface{}) (interface{}, error) {
//	if len(in) == 0 {
//		return map[string]interface{}{}, nil
//	}
//	if fnameUntyped, ok := in["!"]; ok == true {
//		fnameUntyped, err := run(env, fnameUntyped)
//		if err != nil {
//			//TODO structured error
//			return nil, fmt.Errorf("unable to resolve function name: %s", err)
//		}
//		fname, ok := fnameUntyped.(string)
//		if fname == "" || !ok {
//			//TODO structured error
//			return nil, fmt.Errorf("function name is empty or not string: %#v", fname)
//		}
//
//		return "implement map function: " + fname, nil
//	}
//	out := map[string]interface{}{}
//	for k, v := range in {
//		value, err := run(env, v)
//		if err != nil {
//			//TODO structured error
//			return nil, fmt.Errorf("unable to resolve value for key \"%s\": %s", k, err)
//		}
//		out[k] = value
//	}
//	return out, nil
//}
