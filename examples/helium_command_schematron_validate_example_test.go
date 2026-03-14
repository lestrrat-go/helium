package examples_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Example_helium_command_schematron_validate() {
	workDir, err := os.MkdirTemp("", "helium-command-schematron-*")
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
	err = writeHeliumExampleFile(filepath.Join(workDir, "catalog.sch"), `<?xml version="1.0"?>
<schema xmlns="http://www.ascc.net/xml/schematron">
  <pattern>
    <rule context="catalog">
      <assert test="book">catalog must contain at least one book</assert>
    </rule>
  </pattern>
</schema>`)
	if err != nil {
		fmt.Printf("failed to write Schematron schema: %s\n", err)
		return
	}

	stdout, stderr, exitCode := runHeliumCLI("schematron", "validate", filepath.Join(workDir, "catalog.sch"), filepath.Join(workDir, "catalog.xml"))
	if stdout != "" || stderr != "" {
		fmt.Printf("unexpected validation output: stdout=%q stderr=%q\n", strings.TrimSpace(stdout), strings.TrimSpace(stderr))
		return
	}

	fmt.Println("$ helium schematron validate catalog.sch catalog.xml")
	fmt.Printf("exit code: %d\n", exitCode)
	// Output:
	// $ helium schematron validate catalog.sch catalog.xml
	// exit code: 0
}
