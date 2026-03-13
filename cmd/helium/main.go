package main

import (
	"fmt"
	"os"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		showUsage()
		return ExitErr
	}

	switch args[0] {
	case "lint":
		return Run("helium lint", args[1:])
	default:
		fmt.Fprintf(os.Stderr, "helium: unknown subcommand %q\n", args[0])
		showUsage()
		return ExitErr
	}
}

func showUsage() {
	fmt.Fprintln(os.Stderr, `Usage: helium <command> [options]

Available commands:
  lint    Parse and lint XML documents

Planned commands:
  xsd
  xslt`)
}
