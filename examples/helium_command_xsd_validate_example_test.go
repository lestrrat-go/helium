package examples_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Example_helium_command_xsd_validate() {
	workDir, err := os.MkdirTemp("", "helium-command-xsd-*")
	if err != nil {
		fmt.Printf("failed to create temp dir: %s\n", err)
		return
	}
	defer func() { _ = os.RemoveAll(workDir) }()

	err = writeHeliumExampleFile(filepath.Join(workDir, "catalog.xml"), `<?xml version="1.0"?><catalog><book id="b1">Go</book></catalog>`)
	if err != nil {
		fmt.Printf("failed to write XML input: %s\n", err)
		return
	}
	err = writeHeliumExampleFile(filepath.Join(workDir, "catalog.xsd"), `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="catalog">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="book" minOccurs="1" maxOccurs="unbounded">
          <xs:complexType mixed="true">
            <xs:attribute name="id" type="xs:string" use="required"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`)
	if err != nil {
		fmt.Printf("failed to write XSD schema: %s\n", err)
		return
	}

	// Successful schema validation is intentionally quiet, so the example prints
	// the process exit code to show the main success signal callers should use.
	stdout, stderr, exitCode := runHeliumCLI("xsd", "validate", filepath.Join(workDir, "catalog.xsd"), filepath.Join(workDir, "catalog.xml"))
	if stdout != "" || stderr != "" {
		fmt.Printf("unexpected validation output: stdout=%q stderr=%q\n", strings.TrimSpace(stdout), strings.TrimSpace(stderr))
		return
	}

	// The displayed command uses basenames for readability; runHeliumCLI above
	// receives absolute temp-file paths.
	fmt.Println("$ helium xsd validate catalog.xsd catalog.xml")
	fmt.Printf("exit code: %d\n", exitCode)
	// Output:
	// $ helium xsd validate catalog.xsd catalog.xml
	// exit code: 0
}
