package examples_test

import (
	"bytes"
	"context"
	"os"

	"github.com/lestrrat-go/helium/internal/cli/heliumcmd"
)

// runHeliumCLI calls the importable CLI entrypoint directly with injected
// buffers instead of spawning a subprocess. The examples pass absolute paths
// from temporary directories into this helper so they can create isolated test
// fixtures without changing process-wide working directory state; the printed
// shell commands use shorter basenames only to keep the example output readable.
func runHeliumCLI(args ...string) (string, string, int) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	ctx := context.Background()
	ctx = heliumcmd.WithIO(ctx, bytes.NewReader(nil), &stdout, &stderr)
	ctx = heliumcmd.WithStdinTTY(ctx, true)
	exitCode := heliumcmd.Execute(ctx, args)
	return stdout.String(), stderr.String(), exitCode
}

func writeHeliumExampleFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}
