package main

import (
	"fmt"
	"os"

	"github.com/aviason/aviaSurveil/internal/checklistintake"
)

// The parser command is intentionally a bounded worker entrypoint. Archive
// receipt/object ownership remains in the API worker; this binary only proves
// that the pinned sandbox policy is valid before a deployment starts.
func main() {
	if err := checklistintake.DefaultParserSandboxPolicy().Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
