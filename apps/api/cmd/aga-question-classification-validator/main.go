package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	marker, exitCode := runValidatorCommand(os.Args[1:])
	if marker != "" {
		stream := os.Stdout
		if exitCode != 0 {
			stream = os.Stderr
		}
		fmt.Fprintln(stream, marker)
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

// runValidatorCommand emits only controlled, text-free result markers. It
// deliberately accepts a ZIP path rather than a transcript or extracted tree.
func runValidatorCommand(arguments []string) (string, int) {
	if len(arguments) == 0 {
		return diagnostic("ERR_AGA_PASS_INVALID", ""), 2
	}
	switch arguments[0] {
	case "validate-pass":
		return runValidatePass(arguments[1:])
	case "reconcile":
		return runReconcile(arguments[1:])
	case "validate-candidate":
		return runValidateCandidate(arguments[1:])
	default:
		return diagnostic("ERR_AGA_PASS_INVALID", ""), 2
	}
}

func runValidatePass(arguments []string) (string, int) {
	flags := flag.NewFlagSet("validate-pass", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	zipPath := flags.String("zip", "", "untrusted sealed pass ZIP")
	privateRoot := flags.String("private-root", "", "fresh caller-controlled private root")
	expectedPass := flags.String("expected-pass", "", "pass-one or pass-two")
	if flags.Parse(arguments) != nil || flags.NArg() != 0 || *zipPath == "" || *privateRoot == "" {
		return diagnostic("ERR_AGA_PASS_INVALID", ""), 1
	}
	expectedRole := ""
	switch *expectedPass {
	case "pass-one":
		expectedRole = "CANDIDATE"
	case "pass-two":
		expectedRole = "CHALLENGE"
	default:
		return diagnostic("ERR_AGA_PASS_INVALID", ""), 1
	}
	validation, _, err := validatePassZIPInPrivateRoot(*zipPath, *privateRoot, expectedRole)
	if err != nil {
		return diagnostic(err.Error(), ""), 1
	}
	if validation.BatchCount != 25 || validation.RecordCount != 1310 {
		return diagnostic("ERR_AGA_PASS_BIJECTION", ""), 1
	}
	return "AGA_PASS_VALIDATED", 0
}
