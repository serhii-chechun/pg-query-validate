package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/serhii-chechun/pg-query-validate/internal/validate"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr,
			`
PostgreSQL Query Validator v1.0.1 (c) 2026, Serhii Chechun
Usage: pgqv <filename.sql>

`)
		os.Exit(1)
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error closing file: %v\n", err)
		}
	}()

	if err := validate.Process(os.Args[1], file); err != nil {
		if _, ok := errors.AsType[*validate.Error](err); !ok {
			fmt.Fprintf(os.Stderr, "Processing issue: %v\n", err)
		}
		os.Exit(1)
	}
}
