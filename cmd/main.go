package main

import (
	"fmt"
	"os"

	"github.com/unsubble/threadinator/internal/executor"
	"github.com/unsubble/threadinator/internal/parser"
)

func main() {
	args := os.Args[1:]
	config, err := parser.ParseArgs(args)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	if config == nil {
		os.Exit(0)
	}
	err = executor.Execute(config)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
