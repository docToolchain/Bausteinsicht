package main

import "os"

var version = "dev"

func main() {
	rootCmd := NewRootCmd()

	if err := ExecuteRoot(rootCmd); err != nil {
		if e, ok := err.(*exitError); ok {
			os.Exit(e.code)
		}
		os.Exit(1)
	}
}
