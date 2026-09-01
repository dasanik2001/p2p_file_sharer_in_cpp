// cmd/download.go — Download Command
//
// This file defines the `transfera download <invite-code>` subcommand.
// It downloads a file from the server using an invite code shared by the uploader.
//
// The workflow is:
//   1. Parse and validate the invite code (must be a valid port number)
//   2. Stream the file from the server to disk with a progress bar
//   3. Display the download result (filename, size, remaining downloads)
//
// Like the upload command, the actual HTTP work is done by the API client.
// This command handles user interaction: argument parsing, progress display,
// and output formatting.
//
// Usage examples:
//   transfera download 52341
//   transfera download 52341 -o ./received/
//   transfera download 52341 --output-name my-report.pdf

package cmd

import (
	"fmt"     // fmt — formatted printing
	"strconv" // strconv — string-to-number conversion (parsing invite code)

	"github.com/schollz/progressbar/v3" // progressbar — for progress display
	"github.com/spf13/cobra"            // cobra — CLI framework

	"github.com/dasanik2001/transfera-client/cli/internal/api"        // Our API client
	"github.com/dasanik2001/transfera-client/cli/internal/progress"   // Our progress bar
	"github.com/dasanik2001/transfera-client/cli/internal/validation" // For FormatFileSize
)

// ---------------------------------------------------------------------------
// Command-specific flags
// ---------------------------------------------------------------------------

var (
	// outputDir is the directory where the downloaded file will be saved.
	// Default is empty string = current working directory.
	// Example: transfera download 52341 -o ~/Downloads/
	outputDir string

	// outputName overrides the filename from the server.
	// Default is empty string = use the server's original filename.
	// Example: transfera download 52341 --output-name renamed-file.pdf
	outputName string
)

// ---------------------------------------------------------------------------
// Download Command Definition
// ---------------------------------------------------------------------------

var downloadCmd = &cobra.Command{
	Use:   "download <invite-code>",
	Short: "Download a file using an invite code",
	Long: `Download a file from the Transfera server using an invite code.

The invite code is shared by the person who uploaded the file.
Each code has a limited number of downloads (set by the uploader).

The file is streamed directly to disk — only ~128KB of memory is used
regardless of file size.

Examples:
  transfera download 52341
  transfera download 52341 -o ./received/
  transfera download 52341 --output-name my-report.pdf
  transfera --api https://transfera-api.onrender.com download 52341`,

	// Require exactly one argument: the invite code
	Args: cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {
		// =====================================================================
		// Step 1: Parse the invite code
		// =====================================================================
		// The invite code is a port number (integer between 1-65535).
		// strconv.Atoi converts a string to an integer.
		// "Atoi" = "ASCII to integer" (historical C function name).

		inviteCode := args[0]
		port, err := strconv.Atoi(inviteCode)
		if err != nil {
			// The user entered something that isn't a number.
			// Example: transfera download abc
			return fmt.Errorf("invalid invite code '%s': must be a number (1-65535)", inviteCode)
		}

		// Validate the port range — this mirrors the web client's validation
		// in FileDownload.tsx:
		//   if (isNaN(port) || port <= 0 || port > 65535)
		if port <= 0 || port > 65535 {
			return fmt.Errorf("invalid invite code %d: must be between 1 and 65535", port)
		}

		// =====================================================================
		// Step 2: Start the download
		// =====================================================================
		fmt.Printf("\n  Downloading from invite code: %d\n\n", port)

		// Create the API client
		client := api.NewClient(apiBaseURL, verbose)

		// --- Progress bar with lazy initialization ---
		//
		// PROBLEM: We need the total file size to create the progress bar,
		//          but we only learn the size AFTER the server responds.
		//
		// SOLUTION: The API client's Download() accepts a callback
		//           func(bytesReceived, totalBytes). On the FIRST call, we
		//           create the progress bar. On subsequent calls, we update it.
		//
		// We use a *progressbar.ProgressBar pointer, initially nil.
		// The first time the callback is called with a known totalBytes,
		// we create the bar. If totalBytes is unknown (-1), we create
		// an indeterminate spinner instead.
		var pBar *progressbar.ProgressBar

		result, err := client.Download(port, outputDir, outputName, func(received, total int64) {
			// --- Lazy bar creation on first callback ---
			if pBar == nil {
				if total > 0 {
					// We know the total size — create a determinate progress bar
					pBar = progress.NewDownloadBar(total, "file")
				} else {
					// Unknown size — create an indeterminate bar (spinner mode).
					// Use -1 as total to trigger spinner behavior.
					pBar = progress.NewDownloadBar(-1, "file")
				}
			}

			// Update the progress bar with bytes received so far
			pBar.Set64(received)
		})

		if err != nil {
			fmt.Printf("\n")
			return fmt.Errorf("download failed: %w", err)
		}

		// Ensure the bar shows completion
		if pBar != nil {
			pBar.Finish()
		}

		// =====================================================================
		// Step 3: Display the result
		// =====================================================================
		// This is the equivalent of the download success state in the web UI.

		fmt.Printf("\n  ✓ Downloaded: %s (%s)\n",
			result.Filename,
			validation.FormatFileSize(result.FileSize),
		)

		// Show remaining downloads if the server told us.
		// The web UI shows this via the X-Downloads-Remaining header.
		if result.DownloadsRemaining >= 0 {
			if result.DownloadsRemaining == 0 {
				fmt.Printf("    This was the last allowed download. Invite code is now expired.\n")
			} else {
				fmt.Printf("    Downloads remaining: %d\n", result.DownloadsRemaining)
			}
		}

		fmt.Printf("    Saved to: %s\n\n", result.FilePath)

		return nil // Success!
	},
}

// init() registers the download command and its flags.
func init() {
	rootCmd.AddCommand(downloadCmd)

	// -o / --output → directory to save the file in.
	// StringVarP registers a string flag with both long and short names.
	downloadCmd.Flags().StringVarP(
		&outputDir,
		"output", "o",
		"",
		"Output directory (default: current directory)",
	)

	// --output-name → override the filename.
	// StringVar (no P) means there's no short name — only --output-name.
	downloadCmd.Flags().StringVar(
		&outputName,
		"output-name",
		"",
		"Override the download filename",
	)
}
