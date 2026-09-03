// go.mod — Go Module Definition
//
// This file is like package.json in Node.js. It tells Go:
//   1. What this project is called (module path)
//   2. What Go version to use
//   3. What external libraries (dependencies) we need
//
// When you run `go mod tidy`, Go reads this file and downloads
// all dependencies into your local cache.

module github.com/dasanik2001/transfera-client/cli

go 1.22

require (

	// progressbar — Terminal progress bar library
	// Shows upload/download progress with speed, ETA, and percentage.
	// It works with io.Reader/Writer so we can wrap our file streams.
	github.com/schollz/progressbar/v3 v3.17.1
	// cobra — The most popular Go CLI framework (used by kubectl, hugo, github cli)
	// It gives us: subcommands (upload/download/health), flags (--api, --verbose),
	// auto-generated help text, and argument validation.
	github.com/spf13/cobra v1.8.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/sys v0.27.0 // indirect
	golang.org/x/term v0.26.0 // indirect
)
