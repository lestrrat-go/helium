package examples_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Example_helium_command_relaxng_validate() {
	workDir, err := os.MkdirTemp("", "helium-command-relaxng-*")
	if err != nil {
		fmt.Printf("failed to create temp dir: %s\n", err)
		return
	}
	defer func() { _ = os.RemoveAll(workDir) }()

	err = writeHeliumExampleFile(filepath.Join(workDir, "catalog.xml"), `<?xml version="1.0"?><catalog><book>Go</book></catalog>`)
	if err != nil {
		fmt.Printf("failed to write XML input: %s\n", err)
		return
	}
	err = writeHeliumExampleFile(filepath.Join(workDir, "catalog.rng"), `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="catalog">
      <oneOrMore>
        <element name="book">
          <text/>
        </element>
      </oneOrMore>
    </element>
  </start>
</grammar>`)
	if err != nil {
		fmt.Printf("failed to write RELAX NG schema: %s\n", err)
		return
	}

	stdout, stderr, exitCode := runHeliumCLI("relaxng", "validate", filepath.Join(workDir, "catalog.rng"), filepath.Join(workDir, "catalog.xml"))
	if stdout != "" || stderr != "" {
		fmt.Printf("unexpected validation output: stdout=%q stderr=%q\n", strings.TrimSpace(stdout), strings.TrimSpace(stderr))
		return
	}

	fmt.Println("$ helium relaxng validate catalog.rng catalog.xml")
	fmt.Printf("exit code: %d\n", exitCode)
	// Output:
	// $ helium relaxng validate catalog.rng catalog.xml
	// exit code: 0
}
