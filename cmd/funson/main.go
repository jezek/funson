package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/jezek/funson"
)

func main() {
	flag.Usage = func() {
		defer os.Exit(1)
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s SOURCE\n", flag.CommandLine.Name())
		fmt.Fprintf(flag.CommandLine.Output(), "Run funson program in SOURCE and prints result to standart output.\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
	}

	source := flag.Arg(0)

	data, err := ioutil.ReadFile(source)
	if err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Can't read SOURCE file: %s\n", err)
		os.Exit(2)
	}

	var input interface{}
	if err := json.Unmarshal(data, &input); err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Chyba pri citani JSON formatu: %s\n", err)
		os.Exit(3)
	}

	result, err := funson.Fun(input)
	if err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Runtime error: %s\n", err)
		os.Exit(4)
	}

	output, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Marshaling result to JSON error: %s\n", err)
		os.Exit(5)
	}

	fmt.Println(string(output))
}
