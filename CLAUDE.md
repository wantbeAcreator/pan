# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Pan is an Aliyun OSS-based file sync tool. Upload files to an OSS bucket once, download anywhere with `pan down`. Provides both CLI and native GUI (Fyne) interfaces.

## Build Commands

```bash
make              # cross-compile dist/pan-linux-amd64 (default)
make windows      # dist/pan-windows-amd64.exe (needs fyne-cross or mingw-w64)
make darwin-native # dist/pan-darwin-amd64 (native macOS only)
make clean        # remove dist/ and fyne-cross/
```

CGO is required (`CGO_ENABLED=1`) due to Fyne's OpenGL dependencies. The Fyne software renderer is forced via `FYNE_RENDERER=software` in `main.go`.

No tests or linter config exist in this project.

## OSS Configuration

`internal/oss/config.go` is gitignored — create it from `config.go.example` with your Aliyun credentials before building. CI injects credentials from GitHub Secrets at build time.

## Architecture

- **`main.go`** — CLI entry point. Parses args, dispatches `cmdDown()`, `cmdUp()`, `cmdGUI()`. Writes `startup.log` for crash diagnostics.
- **`internal/oss/client.go`** — Core OSS client wrapping `aliyun-oss-go-sdk`. `DownloadAll` uses goroutines + WaitGroup for concurrent downloads. Progress-reporting variants accept callback functions.
- **`internal/oss/config.go`** — Hardcoded credentials (Endpoint, AccessKeyID, AccessKeySecret, BucketName). Single-user tool design.
- **`internal/gui/`** — Fyne native desktop GUI. `app.go` creates 900x600 window with toolbar and file manager. `browser.go` is a table-based file browser with directory navigation, multi-select, and sorting (directories first).
- **`internal/dialog/`** — Platform-specific error dialogs: stderr on Unix, native MessageBox via user32.dll on Windows.
- **`internal/osdetect/`** — Thin wrapper around `runtime.GOOS`.

## OSS Bucket Structure

```
oss://bucket/
  windows/   ← pan down on Windows fetches this prefix
  linux/     ← pan down on Linux fetches this prefix
```

Files are auto-prefixed by OS on upload. `pan down --os <os>` overrides the auto-detection.

## CLI Commands

```
pan down                  # download all files for current OS to cwd
pan down --os windows     # download files for a specific OS
pan up <file>             # upload a file (auto-prefixed by OS)
pan up --os <os> <file>   # upload to a specific OS prefix
pan gui                   # open native desktop file manager
```

## Key Design Decisions

- Go single binary, no runtime dependencies
- Credentials hardcoded (single-user tool)
- Downloads to cwd, always overwrites
- Software renderer forced for GUI compatibility across environments
