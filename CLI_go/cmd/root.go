// cmd/root.go — Root Command & Global Configuration
//
// This file sets up the "parent" command that all subcommands (upload, download,
// health) attach to. Think of it as the "main menu" of the CLI.
//
// When a user types just `transfera`, this root command runs and shows help.
// When they type `transfera upload ...`, cobra routes to the upload subcommand.
//
// GLOBAL FLAGS defined here are available to ALL subcommands:
//   --api <url>     → which server to talk to
//   --verbose       → show extra debug output

package cmd

import (
	"fmt" // fmt = "format" — Go's string formatting & printing (like printf in C)
	"os"  // os = operating system — gives us os.Exit() to quit with an error code

	"github.com/spf13/cobra" // cobra = the CLI framework
)

// ---------------------------------------------------------------------------
// Global variables — accessible by all subcommands (upload.go, download.go, etc.)
// ---------------------------------------------------------------------------

var (
	// apiBaseURL holds the server address. Every HTTP request the CLI makes
	// will be sent to this URL. Default is localhost:8080 (the C++ server's
	// default port). Users override with: transfera --api https://example.com upload ...
	apiBaseURL string

	// verbose controls whether we print extra debug information.
	// When false (default), the CLI only shows essential output.
	// When true (--verbose flag), it prints HTTP request details, timing, etc.
	verbose bool
)

// ---------------------------------------------------------------------------
// Root Command Definition
// ---------------------------------------------------------------------------

// rootCmd is the base command when called without any subcommands.
// cobra.Command is a struct with many fields; we only set the ones we need:
//
//   Use:   — the one-word name shown in usage text
//   Short: — brief description shown in the parent's help
//   Long:  — detailed description shown when user types `transfera --help`
var rootCmd = &cobra.Command{
	Use:   "transfera",
	Short: "Transfera CLI — Secure P2P File Sharing",

	// Long is a multi-line description shown when the user runs `transfera`
	// with no subcommand. The backtick ` allows multi-line strings in Go.
	Long: `
  ___________                          _____                    
 |           |                        / ____|                   
 |--   ------'_ __ __ _ _ __  ___  | |__ ___ _ __ __ _        
    |  |    | '__/ _' | '_ \/ __|  |  __/ _ \ '__/ _' |       
    |  |    | | | (_| | | | \__ \  | | |  __/ | | (_| |       
    |__|    |_|  \__,_|_| |_|___/  |_|  \___|_|  \__,_|       

  Transfera CLI — Secure P2P File Sharing from the terminal.

  Upload files, share invite codes, and download — all without a browser.
  Optimized for large file transfers (100MB+) with minimal memory usage.

  Usage:
    transfera upload <file>           Upload a file and get an invite code
    transfera download <invite-code>  Download a file using an invite code
    transfera health                  Check if the API server is reachable

  The CLI talks to the same C++ API server as the web UI.
  Default server: http://127.0.0.1:8080 (override with --api flag)`,

	// SilenceUsage: when a command returns an error, cobra normally prints
	// the usage text again. We silence this because our error messages are
	// already descriptive enough — showing usage again would be noise.
	SilenceUsage: true,

	// SilenceErrors: we handle error printing ourselves in Execute(),
	// so we tell cobra not to print errors automatically.
	SilenceErrors: true,
}

// ---------------------------------------------------------------------------
// Execute — called by main.go
// ---------------------------------------------------------------------------

// Execute runs the root command. This is the entry point for the entire CLI.
//
// How cobra works internally:
//   1. It reads os.Args (the command-line arguments)
//   2. It finds which subcommand matches (upload, download, health)
//   3. It parses flags (--api, --verbose, --max-downloads, etc.)
//   4. It calls that subcommand's RunE function
//   5. If RunE returns an error, we print it and exit with code 1
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Print the error in red-ish formatting to stderr.
		// os.Stderr is the error output stream (separate from normal output).
		// In terminals, stderr is usually shown even when stdout is redirected.
		fmt.Fprintf(os.Stderr, "\n  ✗ Error: %s\n\n", err)

		// os.Exit(1) terminates the program with exit code 1.
		// Exit code 0 = success, anything else = failure.
		// This lets shell scripts check: `transfera upload ... && echo "success"`
		os.Exit(1)
	}
}

// ---------------------------------------------------------------------------
// init() — runs automatically when this package is loaded
// ---------------------------------------------------------------------------

// init() is a special Go function that runs BEFORE main(). Every package
// can have an init() function. Go guarantees all init() functions run before
// main() starts. We use it to register global flags.
func init() {
	// PersistentFlags are "global" — they work on the root command AND all
	// subcommands. This is different from Flags() which only works on the
	// specific command.
	//
	// StringVarP registers a string flag:
	//   &apiBaseURL  — pointer to the variable where the value is stored
	//   "api"        — the long flag name (--api)
	//   "a"          — the short flag name (-a)
	//   "http://..." — the default value
	//   "API server" — description shown in --help
	rootCmd.PersistentFlags().StringVarP(
		&apiBaseURL,
		"api", "a",
		"http://127.0.0.1:8080",
		"API server URL (e.g., https://transfera-api.onrender.com)",
	)

	// BoolVarP registers a boolean flag:
	//   &verbose — pointer to the bool variable
	//   "verbose" / "V" — long/short names
	//   false — default value (off)
	rootCmd.PersistentFlags().BoolVarP(
		&verbose,
		"verbose", "V",
		false,
		"Enable verbose output (show HTTP details, timing)",
	)
}
