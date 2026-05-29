package main

import (
	"fmt"
	"os"
	"path/filepath"

	"pan/internal/gui"
	"pan/internal/osdetect"
	"pan/internal/oss"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "down":
		cmdDown()
	case "up":
		cmdUp()
	case "gui":
		cmdGUI()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("usage:")
	fmt.Println("  pan down          download all files for current OS")
	fmt.Println("  pan down --os <os>  download files for a specific OS (windows/linux)")
	fmt.Println("  pan gui          open desktop file manager")
	fmt.Println("  pan up <file>     upload a file for current OS")
	fmt.Println("  pan up --os <os> <file>  upload a file for a specific OS")
}

func cmdDown() {
	targetOS := osdetect.CurrentOS()
	args := os.Args[2:]

	if len(args) >= 2 && args[0] == "--os" {
		targetOS = args[1]
	}

	client, err := oss.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	prefix := targetOS + "/"
	if err := client.DownloadAll(prefix, "."); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func cmdUp() {
	targetOS := osdetect.CurrentOS()
	args := os.Args[2:]

	if len(args) >= 2 && args[0] == "--os" {
		targetOS = args[1]
		args = args[2:]
	}

	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "missing file argument")
		printUsage()
		os.Exit(1)
	}

	client, err := oss.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	localPath := args[0]
	remoteKey := targetOS + "/" + filepath.Base(localPath)

	if err := client.Upload(localPath, remoteKey); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func cmdGUI() {
	fmt.Println("starting Pan GUI...")
	if err := gui.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
