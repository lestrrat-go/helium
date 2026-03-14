package examples_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Example_helium_command_xpath() {
	workDir, err := os.MkdirTemp("", "helium-command-xpath-*")
	if err != nil {
		fmt.Printf("failed to create temp dir: %s\n", err)
		return
	}
	defer func() { _ = os.RemoveAll(workDir) }()

	xmlPath := filepath.Join(workDir, "catalog.xml")
	err = writeHeliumExampleFile(xmlPath, `<?xml version="1.0"?><catalog><book>Go</book><book>XML</book></catalog>`)
	if err != nil {
		fmt.Printf("failed to write XML input: %s\n", err)
		return
	}

	// `helium xpath` evaluates an XPath expression and prints the result to
	// stdout. Here we explicitly request engine 1 to show how callers can switch
	// engines when they need compatibility behavior.
	stdout, stderr, exitCode := runHeliumCLI("xpath", "--engine", "1", "count(/catalog/book)", xmlPath)
	if exitCode != 0 || stderr != "" {
		fmt.Printf("unexpected xpath failure: exit=%d stderr=%q\n", exitCode, strings.TrimSpace(stderr))
		return
	}

	fmt.Println("$ helium xpath --engine 1 'count(/catalog/book)' catalog.xml")
	fmt.Println(strings.TrimSpace(stdout))
	// Output:
	// $ helium xpath --engine 1 'count(/catalog/book)' catalog.xml
	// 2
}
