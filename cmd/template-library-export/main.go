package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"menu-service/internal/modules/templatecenter"
)

func main() {
	input := flag.String("input", "", "path to TEMPLATE_LIBRARY_DOC.md")
	output := flag.String("output", "", "path to output json")
	flag.Parse()

	if *input == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "usage: template-library-export -input <markdown> -output <json>")
		os.Exit(2)
	}

	markdown, err := os.ReadFile(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read input: %v\n", err)
		os.Exit(1)
	}
	library, err := templatecenter.ParseTemplateLibraryMarkdown(string(markdown))
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse markdown: %v\n", err)
		os.Exit(1)
	}
	payload, err := json.MarshalIndent(library, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal json: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir output: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*output, append(payload, '\n'), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write output: %v\n", err)
		os.Exit(1)
	}
}
