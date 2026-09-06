// internal/tui/menu.go — Interactive Terminal UI
//
// When the user types `transfera` without any subcommand, or double-clicks
// transfera.exe in Windows Explorer, this interactive menu is displayed.
//
// It provides a friendly, menu-driven interface for all Transfera features:
//   1. Upload a file
//   2. Download a file
//   3. Install to PATH
//   4. Help
//   5. Exit

package tui

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"


	"github.com/dasanik2001/transfera-client/cli/internal/api"
	"github.com/dasanik2001/transfera-client/cli/internal/installer"
	"github.com/dasanik2001/transfera-client/cli/internal/progress"
	"github.com/dasanik2001/transfera-client/cli/internal/validation"
	"github.com/schollz/progressbar/v3"
)

// =========================================================================
// Color helpers — ANSI escape codes for terminal styling
// =========================================================================

const (
	colorReset   = "\033[0m"
	colorCyan    = "\033[36m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorRed     = "\033[31m"
	colorBold    = "\033[1m"
	colorDim     = "\033[2m"
	colorWhite   = "\033[97m"
	colorMagenta = "\033[35m"
)

// =========================================================================
// RunMenu — entry point for the interactive TUI
// =========================================================================

// RunMenu displays the interactive terminal menu and handles user actions.
// apiURL and verbose are pointers to the global config from root.go.
func RunMenu(apiURL *string, verbose *bool) {
	reader := bufio.NewReader(os.Stdin)

	clearScreen()
	printBanner()

	for {
		printMenu()
		choice := prompt(reader, fmt.Sprintf("\n  %s▶ Choose an option (1-5):%s ", colorCyan, colorReset))

		switch strings.TrimSpace(choice) {
		case "1":
			handleUpload(reader, *apiURL, *verbose)
		case "2":
			handleDownload(reader, *apiURL, *verbose)
		case "3":
			handleInstall(reader)
		case "4":
			handleHelp()
		case "5", "q", "Q", "exit", "quit":
			fmt.Printf("\n  %s👋 Goodbye!%s\n\n", colorCyan, colorReset)
			return
		default:
			fmt.Printf("\n  %s✗ Invalid option. Please enter 1-5.%s\n", colorRed, colorReset)
		}

		fmt.Println()
		promptContinue(reader)
	}
}

// =========================================================================
// Banner & Menu Display
// =========================================================================

func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

func printBanner() {
	fmt.Println()
	fmt.Print(colorCyan, colorBold)
	fmt.Println("   ___________                          _____")
	fmt.Println("  |           |                        / ____|")
	fmt.Println("  |--   ------'_ __ __ _ _ __  ___  | |__ ___ _ __ __ _")
	fmt.Println("     |  |    | '__/ _' | '_ \\/ __|  |  __/ _ \\ '__/ _' |")
	fmt.Println("     |  |    | | | (_| | | | \\__ \\  | | |  __/ | | (_| |")
	fmt.Println("     |__|    |_|  \\__,_|_| |_|___/  |_|  \\___|_|  \\__,_|")
	fmt.Print(colorReset)
	fmt.Println()
	fmt.Printf("  %s%sSecure P2P File Sharing — Terminal Edition%s\n", colorDim, colorWhite, colorReset)
	fmt.Println()

	// PATH installation status
	if installer.IsInPath() && installer.IsInstalled() {
		fmt.Printf("  %s● Status:%s %s✓ Installed globally%s — type %stransfera%s anywhere\n",
			colorDim, colorReset, colorGreen, colorReset, colorBold, colorReset)
	} else {
		fmt.Printf("  %s● Status:%s %s⚠ Not installed to PATH%s — select option 3 to install\n",
			colorDim, colorReset, colorYellow, colorReset)
	}
	fmt.Println()
}

func printMenu() {
	fmt.Printf("  %s%s┌──────────────────────────────────────────────┐%s\n", colorCyan, colorBold, colorReset)
	fmt.Printf("  %s%s│%s   %-43s%s%s│%s\n", colorCyan, colorBold, colorReset, "What would you like to do?", colorCyan, colorBold, colorReset)
	fmt.Printf("  %s%s├──────────────────────────────────────────────┤%s\n", colorCyan, colorBold, colorReset)
	fmt.Printf("  %s%s│%s   %s1.%s 📤  Upload a file                       %s%s│%s\n", colorCyan, colorBold, colorReset, colorGreen, colorReset, colorCyan, colorBold, colorReset)
	fmt.Printf("  %s%s│%s   %s2.%s 📥  Download a file                     %s%s│%s\n", colorCyan, colorBold, colorReset, colorGreen, colorReset, colorCyan, colorBold, colorReset)
	fmt.Printf("  %s%s│%s   %s3.%s ⚡  Install to PATH                     %s%s│%s\n", colorCyan, colorBold, colorReset, colorYellow, colorReset, colorCyan, colorBold, colorReset)
	fmt.Printf("  %s%s│%s   %s4.%s ❓  Help & CLI Reference                %s%s│%s\n", colorCyan, colorBold, colorReset, colorDim, colorReset, colorCyan, colorBold, colorReset)
	fmt.Printf("  %s%s│%s   %s5.%s ❌  Exit                                %s%s│%s\n", colorCyan, colorBold, colorReset, colorRed, colorReset, colorCyan, colorBold, colorReset)
	fmt.Printf("  %s%s└──────────────────────────────────────────────┘%s\n", colorCyan, colorBold, colorReset)
}

// =========================================================================
// Menu Handlers
// =========================================================================

func handleUpload(reader *bufio.Reader, apiURL string, verbose bool) {
	fmt.Printf("\n  %s%s── Upload a File ──%s\n\n", colorCyan, colorBold, colorReset)

	// Prompt for file path (supports drag-and-drop: strips quotes)
	filePath := prompt(reader, fmt.Sprintf("  %sFile path (drag & drop or type):%s ", colorWhite, colorReset))
	filePath = cleanPath(filePath)

	if filePath == "" {
		fmt.Printf("  %s✗ No file specified.%s\n", colorRed, colorReset)
		return
	}

	// Validate file
	fileSize, err := validation.ValidateFile(filePath, validation.DefaultMaxUploadMB)
	if err != nil {
		fmt.Printf("  %s✗ %s%s\n", colorRed, err, colorReset)
		return
	}

	// Prompt for max downloads
	maxDlStr := prompt(reader, fmt.Sprintf("  %sMax downloads (1-100, default 1):%s ", colorWhite, colorReset))
	maxDownloads := 1
	if maxDlStr != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(maxDlStr)); err == nil && n >= 1 && n <= 100 {
			maxDownloads = n
		}
	}

	filename := filepath.Base(filePath)
	fmt.Printf("\n  File: %s%s%s (%s)\n", colorBold, filename, colorReset, validation.FormatFileSize(fileSize))
	fmt.Printf("  Max downloads: %d\n\n", maxDownloads)

	// Create progress bar and upload
	bar := progress.NewUploadBar(fileSize, filename)
	client := api.NewClient(apiURL, verbose)

	result, err := client.Upload(filePath, maxDownloads, func(bytesSent int64) {
		bar.Set64(bytesSent)
	})

	if err != nil {
		fmt.Printf("\n  %s✗ Upload failed: %s%s\n", colorRed, err, colorReset)
		return
	}

	bar.Finish()

	// Display invite code
	fmt.Printf("\n  %s✓ File ready to share!%s\n", colorGreen, colorReset)
	fmt.Printf("  %s┌────────────────────────────────────────%s\n", colorCyan, colorReset)
	fmt.Printf("  %s│%s  Invite code: %s%s%d%s\n", colorCyan, colorReset, colorBold, colorGreen, result.Port, colorReset)
	fmt.Printf("  %s│%s  Max downloads: %d\n", colorCyan, colorReset, result.MaxDownloads)
	fmt.Printf("  %s└────────────────────────────────────────%s\n", colorCyan, colorReset)
	fmt.Printf("\n  Share this code to download on another device:\n")
	fmt.Printf("    %stransfera download %d%s\n", colorBold, result.Port, colorReset)
}

func handleDownload(reader *bufio.Reader, apiURL string, verbose bool) {
	fmt.Printf("\n  %s%s── Download a File ──%s\n\n", colorCyan, colorBold, colorReset)

	// Prompt for invite code
	codeStr := prompt(reader, fmt.Sprintf("  %sInvite code:%s ", colorWhite, colorReset))
	codeStr = strings.TrimSpace(codeStr)

	port, err := strconv.Atoi(codeStr)
	if err != nil || port <= 0 || port > 65535 {
		fmt.Printf("  %s✗ Invalid invite code '%s'. Must be a number (1-65535).%s\n", colorRed, codeStr, colorReset)
		return
	}

	// Prompt for output directory
	outputDir := prompt(reader, fmt.Sprintf("  %sSave to directory (Enter = current dir):%s ", colorWhite, colorReset))
	outputDir = cleanPath(outputDir)

	fmt.Printf("\n  Downloading from invite code: %s%d%s\n\n", colorBold, port, colorReset)

	client := api.NewClient(apiURL, verbose)
	var pBar *progressbar.ProgressBar

	result, err := client.Download(port, outputDir, "", func(filename string, received, total int64) {
		if pBar == nil {
			displayName := filename
			if displayName == "" {
				displayName = "file"
			}
			if total > 0 {
				pBar = progress.NewDownloadBar(total, displayName)
			} else {
				pBar = progress.NewDownloadBar(-1, displayName)
			}
		}
		pBar.Set64(received)
	})

	if err != nil {
		fmt.Printf("\n  %s✗ Download failed: %s%s\n", colorRed, err, colorReset)
		return
	}

	if pBar != nil {
		pBar.Finish()
	}

	fmt.Printf("\n  %s✓ Downloaded: %s%s (%s)\n", colorGreen, result.Filename, colorReset,
		validation.FormatFileSize(result.FileSize))

	if result.DownloadsRemaining >= 0 {
		if result.DownloadsRemaining == 0 {
			fmt.Printf("    %sThis was the last allowed download. Invite code is now expired.%s\n", colorYellow, colorReset)
		} else {
			fmt.Printf("    Downloads remaining: %d\n", result.DownloadsRemaining)
		}
	}
	fmt.Printf("    Saved to: %s%s%s\n", colorBold, result.FilePath, colorReset)
}

func handleInstall(reader *bufio.Reader) {
	destPath := installer.InstalledBinaryPath()
	isInstalled := installer.IsInstalled()
	isInPath := installer.IsInPath()
	hasUpdate := isInstalled && installer.IsUpdateAvailable()

	if isInstalled && isInPath {
		if hasUpdate {
			fmt.Printf("\n  %s%s── Update Transfera in PATH ──%s\n\n", colorCyan, colorBold, colorReset)
			fmt.Printf("  %s🔄 New version / updated binary detected!%s\n", colorYellow, colorReset)
			fmt.Printf("    Installed: %s\n\n", destPath)
		} else {
			fmt.Printf("\n  %s%s── Reinstall / Update in PATH ──%s\n\n", colorCyan, colorBold, colorReset)
			fmt.Printf("  %s✓ Transfera is already installed and in PATH.%s\n", colorGreen, colorReset)
			fmt.Printf("    Installed: %s\n\n", destPath)
		}

		confirm := prompt(reader, fmt.Sprintf("  %s▶ Overwrite/reinstall binary? (Y/n):%s ", colorCyan, colorReset))
		confirm = strings.ToLower(strings.TrimSpace(confirm))
		if confirm != "" && confirm != "y" && confirm != "yes" {
			fmt.Printf("\n  %sNo changes made.%s\n", colorDim, colorReset)
			return
		}
		fmt.Println()
	} else {
		fmt.Printf("\n  %s%s── Install to PATH ──%s\n\n", colorCyan, colorBold, colorReset)
	}

	fmt.Printf("  Target directory: %s\n", installer.InstallDir())
	fmt.Printf("  Installing binary & verifying PATH...\n\n")

	msg, err := installer.Install()
	if err != nil {
		fmt.Printf("  %s✗ Installation failed: %s%s\n", colorRed, err, colorReset)
		return
	}

	fmt.Printf("  %s✓ %s%s\n", colorGreen, msg, colorReset)
}

func handleHelp() {
	fmt.Printf("\n  %s%s── CLI Quick Reference ──%s\n\n", colorCyan, colorBold, colorReset)
	fmt.Printf("  %sUpload a file:%s\n", colorBold, colorReset)
	fmt.Printf("    transfera upload photo.jpg\n")
	fmt.Printf("    transfera upload video.mp4 --max-downloads 5\n\n")
	fmt.Printf("  %sDownload a file:%s\n", colorBold, colorReset)
	fmt.Printf("    transfera download 52341\n")
	fmt.Printf("    transfera download 52341 -o ~/Pictures/\n\n")
	fmt.Printf("  %sHealth check:%s\n", colorBold, colorReset)
	fmt.Printf("    transfera health\n\n")
	fmt.Printf("  %sGlobal flags:%s\n", colorBold, colorReset)
	fmt.Printf("    --api <url>    Set API server URL\n")
	fmt.Printf("    --verbose      Show HTTP details\n\n")
	fmt.Printf("  %sEnvironment variables:%s\n", colorBold, colorReset)
	fmt.Printf("    TRANSFERA_API_URL         Override default server\n")
	fmt.Printf("    NEXT_PUBLIC_API_BASE_URL  Alternative override\n")
}

// =========================================================================
// Utility functions
// =========================================================================

func prompt(reader *bufio.Reader, msg string) string {
	fmt.Print(msg)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(strings.TrimRight(line, "\r\n"))
}

func promptContinue(reader *bufio.Reader) {
	fmt.Printf("  %sPress Enter to continue...%s", colorDim, colorReset)
	reader.ReadString('\n')
	clearScreen()
}

// cleanPath strips surrounding quotes, whitespace, expands tildes (~),
// and unescapes backslash-escaped spaces from terminal drag-and-drop on macOS/Linux.
func cleanPath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.Trim(p, "\"'")

	// If file exists as-is, return it
	if _, err := os.Stat(p); err == nil {
		return p
	}

	// Expand tilde (~)
	if p == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	} else if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, p[2:])
		}
	}

	// Handle terminal drag-and-drop on macOS/Linux escaping spaces with "\ "
	if strings.Contains(p, `\ `) {
		unescaped := strings.ReplaceAll(p, `\ `, " ")
		if _, err := os.Stat(unescaped); err == nil {
			return unescaped
		}
		p = unescaped
	}

	return strings.TrimSpace(p)
}
