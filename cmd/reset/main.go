package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: go run ./cmd/reset <paths...>")
		fmt.Fprintln(os.Stderr, "Example: go run ./cmd/reset ./internal/... ./cmd/...")
		os.Exit(1)
	}

	paths := os.Args[1:]

	gen := NewGenerator()

	if err := gen.ScanPackages(paths); err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning packages: %v\n", err)
		os.Exit(1)
	}

	structs := gen.FindMarkedStructs()

	if len(structs) == 0 {
		fmt.Println("No structs marked with // generate:reset found")
		return
	}

	if err := gen.GenerateCode(structs); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating code: %v\n", err)
		os.Exit(1)
	}

	for path, name := range gen.GetGeneratedFiles() {
		fmt.Printf("Generated %s for %s\n", path, name)
	}
}