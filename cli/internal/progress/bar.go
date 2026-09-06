// internal/progress/bar.go — Terminal Progress Bar
//
// This file creates beautiful terminal progress bars for uploads and downloads.
// It wraps the progressbar library with our preferred styling.
//
// WHY a wrapper?
//   - Consistent look: both upload and download bars look the same
//   - Single place to change styling (colors, format, speed units)
//   - The progressbar library has MANY options — this simplifies usage
//
// WHAT IT LOOKS LIKE:
//   Uploading video.mp4
//    98% |████████████████████████████████████████████   | (98/100 MB, 12.3 MB/s) [8s:0s]

package progress

import (
	"fmt" // fmt — formatting (building the description string)

	// The progressbar library. v3 is the latest version.
	// It draws a progress bar in the terminal and updates it in-place
	// (using \r carriage return to overwrite the same line).
	"github.com/schollz/progressbar/v3"
)

// =========================================================================
// NewBar — create a progress bar for any file operation
// =========================================================================

// NewBar creates a new progress bar for tracking file transfer progress.
//
// Parameters:
//   totalBytes  — total file size in bytes (for calculating percentage)
//   description — text shown before the bar (e.g., "Uploading video.mp4")
//
// Returns:
//   *progressbar.ProgressBar — the bar object. Call bar.Add(n) to advance it.
//
// HOW IT WORKS:
//   1. The bar is created with the total byte count
//   2. As chunks are sent/received, we call bar.Set64(bytesSoFar)
//   3. The library calculates percentage, speed, and ETA automatically
//   4. It redraws the bar on the same terminal line using \r
func NewBar(totalBytes int64, description string) *progressbar.ProgressBar {
	return progressbar.NewOptions64(
		totalBytes, // The total value (100% = totalBytes)

		// --- Visual options ---

		// SetDescription: text shown before the progress bar
		// Example: "  Uploading video.mp4"
		// The leading spaces indent it for nicer formatting.
		progressbar.OptionSetDescription(fmt.Sprintf("  %s", description)),

		// SetWidth: the character width of the bar itself (not counting labels).
		// 40 characters gives a good balance between bar size and info display.
		progressbar.OptionSetWidth(40),

		// ShowBytes: display progress in human-readable bytes
		// Instead of "52428800/104857600" it shows "50 MB/100 MB"
		progressbar.OptionShowBytes(true),

		// ShowCount: show the numeric progress (current/total)
		progressbar.OptionShowCount(),

		// SetTheme: customize the bar characters
		// Saucer     = filled portion (█)
		// SaucerHead = the leading edge of filled portion (█)
		// SaucerPadding = unfilled portion (░)
		// BarStart/BarEnd = the brackets around the bar
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "█",
			SaucerHead:    "█",
			SaucerPadding: "░",
			BarStart:      "|",
			BarEnd:        "|",
		}),

		// OnCompletion: called when progress reaches 100%.
		// We print a newline so the next output starts on a fresh line
		// (otherwise it would be appended to the progress bar line).
		progressbar.OptionOnCompletion(func() {
			fmt.Println()
		}),

		// ClearOnFinish: DON'T clear the bar when done.
		// We want the user to see the completed bar with final stats.
		progressbar.OptionClearOnFinish(),

		// SpinnerType: the animation style for indeterminate progress.
		// Type 14 is a smooth spinning animation: ⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏
		// This is used when totalBytes is unknown.
		progressbar.OptionSpinnerType(14),

		// SetPredictTime: show estimated time remaining based on current speed.
		// Example: [8s:2s] means "8 seconds elapsed, ~2 seconds remaining"
		progressbar.OptionSetPredictTime(true),

		// SetElapsedTime: show how long the transfer has taken so far.
		progressbar.OptionSetElapsedTime(true),
	)
}

// =========================================================================
// NewUploadBar — convenience wrapper for uploads
// =========================================================================

// NewUploadBar creates a progress bar specifically for file uploads.
// It prefixes the filename with "Uploading" for clarity.
//
// Usage:
//   bar := progress.NewUploadBar(fileSize, "video.mp4")
//   bar.Set64(bytesSent)  // called in the upload loop
func NewUploadBar(totalBytes int64, filename string) *progressbar.ProgressBar {
	return NewBar(totalBytes, fmt.Sprintf("Uploading %s", filename))
}

// =========================================================================
// NewDownloadBar — convenience wrapper for downloads
// =========================================================================

// NewDownloadBar creates a progress bar specifically for file downloads.
// It prefixes the filename with "Downloading" for clarity.
//
// Usage:
//   bar := progress.NewDownloadBar(fileSize, "video.mp4")
//   bar.Set64(bytesReceived)  // called in the download loop
func NewDownloadBar(totalBytes int64, filename string) *progressbar.ProgressBar {
	return NewBar(totalBytes, fmt.Sprintf("Downloading %s", filename))
}
