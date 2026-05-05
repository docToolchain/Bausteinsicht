package main

import (
	"flag"
	"log"
	"os"

	"github.com/docToolchain/Bausteinsicht/internal/lsp"
)

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "Enable debug logging to stderr")
	flag.Parse()

	// Set up logging
	logFile := os.Stderr
	if !debug {
		logFile, _ = os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	}
	log.SetOutput(logFile)

	// Create and run LSP server
	server := lsp.NewServer()
	if err := server.Run(); err != nil {
		log.Fatalf("LSP server error: %v", err)
	}
}
