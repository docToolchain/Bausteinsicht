package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	if err := NewRootCmd().Execute(); err != nil {
		if ee, ok := err.(*exitError); ok {
			fmt.Fprintln(os.Stderr, ee.Error())
			os.Exit(ee.code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
