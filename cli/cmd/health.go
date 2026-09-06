// cmd/health.go — Health Check Command
//
// This file defines the `transfera health` subcommand.
// It checks if the C++ API server is running and reachable.
//
// This is the simplest command — a great starting point to understand
// how cobra commands work before looking at upload/download.
//
// Usage:
//   transfera health                        → checks http://127.0.0.1:8080
//   transfera --api https://example.com health → checks custom server

package cmd

import (
	"fmt"     // fmt — formatted output (printing results to terminal)
	"strings" // strings — URL checking

	"github.com/spf13/cobra" // cobra — CLI framework

	// Import our internal API client package.
	// This is the package we created in internal/api/client.go.
	"github.com/dasanik2001/transfera-client/cli/internal/api"
)

// healthCmd defines the `transfera health` subcommand.
//
// cobra.Command fields:
//   Use:   — command name as typed by the user
//   Short: — one-line description shown in parent help
//   Long:  — detailed description shown with `transfera health --help`
//   RunE:  — the function to execute (returns an error, unlike Run which doesn't)
//           We use RunE (not Run) because health checks can fail, and returning
//           an error lets cobra handle it consistently.
var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check if the API server is reachable",
	Long: `Check if the Transfera API server is reachable and responding to requests.

Defaults to the official cloud server: https://transfera-api.onrender.com
Override with --api or TRANSFERA_API_URL environment variable.

Examples:
  transfera health
  transfera --api http://127.0.0.1:8080 health`,

	// RunE is the function that runs when the user types `transfera health`.
	// It receives the command and any positional arguments (though health takes none).
	//
	// The _ before cobra.Command means "I know this parameter exists but I'm
	// not going to use it." Go requires you to acknowledge all parameters.
	// Similarly, _ for args means we ignore positional arguments.
	RunE: func(_ *cobra.Command, _ []string) error {
		// --- Step 1: Create the API client ---
		// apiBaseURL and verbose come from root.go (global variables set by flags).
		// NewClient creates an HTTP client configured with the right URL and timeouts.
		client := api.NewClient(apiBaseURL, verbose)

		// --- Step 2: Call the health endpoint ---
		isLocal := strings.Contains(apiBaseURL, "127.0.0.1") || strings.Contains(apiBaseURL, "localhost")
		if isLocal {
			fmt.Printf("\n  Checking API server at %s ...\n", apiBaseURL)
		} else {
			fmt.Printf("\n  Checking API server at %s (may take a few seconds if waking up)... \n", apiBaseURL)
		}

		ok, err := client.Health()

		// --- Step 3: Display the result ---
		if err != nil {
			// Server is unreachable — show error with fix instructions.
			// The \n at start/end adds blank lines for readability.
			fmt.Printf("\n  ✗ API server is unreachable at %s\n", apiBaseURL)
			fmt.Printf("    %s\n", err)

			if isLocal {
				fmt.Printf("\n  To start the local server:\n")
				fmt.Printf("    cd server && ./scripts/run.sh\n\n")
			} else {
				fmt.Printf("\n  Tip: The cloud server on Render spins down when idle.\n")
				fmt.Printf("  It may take 30-50 seconds to wake up. Please try again shortly!\n\n")
			}

			// Return the error so cobra exits with code 1.
			// This is important for scripting:
			//   transfera health && echo "server is up"
			return fmt.Errorf("API server unreachable")
		}

		if ok {
			// Server is healthy! ✓ symbol shows success visually.
			fmt.Printf("\n  ✓ API server is online at %s\n\n", apiBaseURL)
		} else {
			// Server responded but not with 200 — something is wrong.
			fmt.Printf("\n  ⚠ API server responded but may not be healthy at %s\n\n", apiBaseURL)
		}

		return nil // nil = success, exit code 0
	},
}

// init() registers the health command as a subcommand of the root command.
// Without this, typing `transfera health` would say "unknown command".
//
// Go calls init() automatically when the package is loaded.
// The order: root.go init() runs first (sets up flags), then health.go init()
// runs and adds itself as a child of rootCmd.
func init() {
	rootCmd.AddCommand(healthCmd)
}
