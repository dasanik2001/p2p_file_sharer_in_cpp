// main.go — Application Entry Point
//
// This is the FIRST file Go runs when you execute the CLI binary.
// Its ONLY job is to call cmd.Execute() which starts the cobra CLI framework.
//
// WHY so minimal?
// In Go, the main() function should be as thin as possible. All the real
// logic lives in the `cmd` package. This separation makes the code testable
// — you can test commands without running the whole binary.

package main

import (
	// Import our cmd package — this is where all CLI commands are defined.
	// The path matches our module name + /cmd subfolder.
	"github.com/dasanik2001/transfera-client/cli/cmd"
)

func main() {
	// Execute() is defined in cmd/root.go. It parses the command-line
	// arguments, finds the right subcommand (upload/download/health),
	// and runs it. If anything goes wrong, it prints an error and exits
	// with code 1.
	cmd.Execute()
}
