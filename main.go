package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"pan/internal/dialog"
	"pan/internal/gui"
	"pan/internal/osdetect"
	"pan/internal/oss"
)

var startupLog *os.File

func logWrite(s string) {
	startupLog.WriteString(s)
	startupLog.Sync()
}

func init() {
	var err error
	startupLog, err = os.Create("startup.log")
	if err != nil {
		startupLog = os.Stderr
	}
	logWrite(fmt.Sprintf("=== pan started at %s ===\n", time.Now().Format(time.RFC3339)))

	os.Setenv("FYNE_RENDERER", "software")
}

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

func validateOS(targetOS string) {
	validOS := map[string]bool{"windows": true, "linux": true, "darwin": true}
	if !validOS[targetOS] {
		fmt.Fprintf(os.Stderr, "unsupported OS: %s (valid: windows, linux, darwin)\n", targetOS)
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
	validateOS(targetOS)

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
	validateOS(targetOS)

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
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("panic: %v", r)
			logWrite(msg + "\n")
			dialog.ShowError("Pan Error", msg)
		}
		if startupLog != nil && startupLog != os.Stderr {
			startupLog.Close()
		}
	}()

	logWrite("step1: entering gui.Start()...\n")
	gui.Start()

	if startupLog != nil && startupLog != os.Stderr {
		startupLog.Close()
	}
}
