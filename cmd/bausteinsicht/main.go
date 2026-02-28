package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	rootCmd := NewRootCmd()

	if err := rootCmd.Execute(); err != nil {
		if e, ok := err.(*exitError); ok {
			fmt.Fprintln(os.Stderr, e.Error())
			os.Exit(e.code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
