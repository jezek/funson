# funson
Another approach to functional JSON for fun.

The main idea is that the JSON text is the input to a parser, the same JSON text contains the program and output is the JSON text modified by the program.

In this case, the parser is the ```funson``` executable residing in the ```cmd/funson``` directory.

If a valid JSON file is given to ```funson``` as argument, the JSON tree is parsed. The JSON tree consists of branches and leaves.
Branches are array value (```[ … ]```) and object value (```{ … }```).
All other values (numbers, strings, true, false, null) are leaves.

If the parser encounters an leaf type and currently the branch object it doesn't do any operation upon them and leaves them as they are.
If the parser encounters an array, and the array qualifies as function, the function is run and the output will be inserted into the tree instead of the array. If the array is not a function, all its elements are parsed (from first to last) and the results will make the new array.

The array qualifies as function if its first field is a string and begins with ```!``` followed by any other character.
For example, if the parser encounters an array looking like ```[ "!functionName", 1, true, "foo" ]``` it would attempt to run a function ```functionName``` with the values ```1, true, "foo"``` as parameters. If the ```funson``` program doesn't know such a function or the parameters mismatch (wrong number of parameters or wrong types) it panics and produces no output. Currently the functions you can use are hard-coded and the list can be viewed by exploring the examples in the ```examples``` directory or the code in ```globalFunctions.go```. Maybe in the future I will implement a way to be able to define functions in the JSON tree itself. (TODO make list of available hardcoded functions, or let the ```funson``` to output them.)

All the parameters are parsed before calling the function and the parsed results will be fed to the function.
For example, if the input is ```[ "!f1", 2, [ "!f2", 5.8 ], "bar" ]```, then first the ```f2``` function is called with ```5.8``` as parameter and then the result will replace the ```[ "!f2", 5.8 ]``` array and function ```f1``` will be called. Note: The underlying language is ```go``` which can return more than one result, so the functions in ```funson``` can return multiple results.

## Why?
I had a cli go project and a part of it was to generate JSON from user input, with some predefined choices. Part of the project was to be able to edit the choices and questions and add/remove stuff to/from the JSON, without recompiling the executable. It was a call for some scripting language that could be embedded into the project. I thought to myself, that it would be fun to have the program, that interacts with user and generates the JSON tree, placed in the same JSON tree. It was my project, my calls and I created funson (functional JSON). Later I extracted the funson part from the project into this package, with the thought that maybe it is an interesting concept and someone would like it and maybe help me to expand and/or make the idea better.

## External golang dependencies:
- [github.com/mohae/deepcopy](github.com/mohae/deepcopy)

## Useful links
- [Introducing JSON](https://www.json.org/json-en.html)
