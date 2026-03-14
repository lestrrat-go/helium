package examples_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Example_helium_command_lint() {
	// Each example gets its own temporary directory so it can create isolated
	// on-disk inputs without interfering with other tests.
	workDir, err := os.MkdirTemp("", "helium-command-lint-*")
	if err != nil {
		fmt.Printf("failed to create temp dir: %s\n", err)
		return
	}
	defer func() { _ = os.RemoveAll(workDir) }()

	xmlPath := filepath.Join(workDir, "catalog.xml")
	err = writeHeliumExampleFile(xmlPath, `<?xml version="1.0"?><catalog><book id="b1">Go</book><book id="b2">XML</book></catalog>`)
	if err != nil {
		fmt.Printf("failed to write XML input: %s\n", err)
		return
	}

	// `helium lint --format` parses the document and writes a formatted copy to
	// stdout, which makes it a good first command to show in an example.
	stdout, stderr, exitCode := runHeliumCLI("lint", "--format", xmlPath)
	if exitCode != 0 || stderr != "" {
		fmt.Printf("unexpected lint failure: exit=%d stderr=%q\n", exitCode, strings.TrimSpace(stderr))
		return
	}

	// The displayed command uses a basename for readability; runHeliumCLI above
	// receives the absolute temp-file path.
	fmt.Println("$ helium lint --format catalog.xml")
	fmt.Println(strings.TrimRight(stdout, "\n"))
	// Output:
	// $ helium lint --format catalog.xml
	// <?xml version="1.0"?>
	// <catalog>
	//   <book id="b1">Go</book>
	//   <book id="b2">XML</book>
	// </catalog>
}
