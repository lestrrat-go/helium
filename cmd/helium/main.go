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
	case "relaxng":
		return runRelaxNG(args[1:])
	case "schematron":
		return runSchematron(args[1:])
	case "xpath":
		return RunXPath("helium xpath", args[1:])
	case "xsd":
		return runXSD(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "helium: unknown subcommand %q\n", args[0])
		showUsage()
		return ExitErr
	}
}

func runRelaxNG(args []string) int {
	if len(args) == 0 {
		showRelaxNGUsage()
		return ExitErr
	}

	switch args[0] {
	case "validate":
		return RunRelaxNGValidate("helium relaxng validate", args[1:])
	default:
		fmt.Fprintf(os.Stderr, "helium relaxng: unknown subcommand %q\n", args[0])
		showRelaxNGUsage()
		return ExitErr
	}
}

func runSchematron(args []string) int {
	if len(args) == 0 {
		showSchematronUsage()
		return ExitErr
	}

	switch args[0] {
	case "validate":
		return RunSchematronValidate("helium schematron validate", args[1:])
	default:
		fmt.Fprintf(os.Stderr, "helium schematron: unknown subcommand %q\n", args[0])
		showSchematronUsage()
		return ExitErr
	}
}

func runXSD(args []string) int {
	if len(args) == 0 {
		showXSDUsage()
		return ExitErr
	}

	switch args[0] {
	case "validate":
		return RunXSDValidate("helium xsd validate", args[1:])
	default:
		fmt.Fprintf(os.Stderr, "helium xsd: unknown subcommand %q\n", args[0])
		showXSDUsage()
		return ExitErr
	}
}

func showUsage() {
	fmt.Fprintln(os.Stderr, `Usage: helium <command> [options]

Available commands:
  lint    Parse and lint XML documents
  relaxng RELAX NG operations
  schematron Schematron operations
  xpath   Evaluate XPath expressions
  xsd     XML Schema operations

Planned commands:
  xslt`)
}

func showXSDUsage() {
	fmt.Fprintln(os.Stderr, `Usage: helium xsd <command> [options]

Available commands:
  validate    Validate XML documents against an XML Schema`)
}

func showRelaxNGUsage() {
	fmt.Fprintln(os.Stderr, `Usage: helium relaxng <command> [options]

Available commands:
  validate    Validate XML documents against a RELAX NG schema`)
}

func showSchematronUsage() {
	fmt.Fprintln(os.Stderr, `Usage: helium schematron <command> [options]

Available commands:
  validate    Validate XML documents against a Schematron schema`)
}
