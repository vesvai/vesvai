package main

import (
	"fmt"
	"os"

	"github.com/vesvai/vesvai/internal/core/bootstrap"
)

func main() {
	if err := bootstrap.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "vesvai: %v\n", err)
		os.Exit(1)
	}
}
