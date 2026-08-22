package main

import (
	"flag"
	"fmt"
	"io"
)

func flagSet(name string, stderr io.Writer) *flag.FlagSet {
	set := flag.NewFlagSet(name, flag.ContinueOnError)
	set.SetOutput(stderr)
	return set
}

func readAll(reader io.Reader) (string, error) {
	content, err := io.ReadAll(reader)
	return string(content), err
}

func fail(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "deja: %v\n", err)
	return 2
}
