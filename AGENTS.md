# AGENTS.md

## Project

Aliyun OSS-based file sync tool. Upload software to OSS once, download everywhere with `pan down`.

## Build

```bash
make
```

Cross-compiles `dist/pan-linux-amd64` and `dist/pan-windows-amd64.exe`.

## Architecture

- `main.go` — CLI entry, dispatches `pan down` / `pan up` / `pan gui`
- `internal/oss/` — OSS client: ListAll, DownloadAll (concurrent), DownloadFile, Upload
- `internal/osdetect/` — wraps `runtime.GOOS`
- `internal/oss/config.go` — hardcoded credentials (single-user tool)
- `internal/webui/` — embedded web UI for `pan gui` (SSE progress, browser-based)

## OSS structure

```
oss://bucket/
  windows/  ← pan down on Windows downloads everything under this prefix
  linux/    ← pan down on Linux downloads everything under this prefix
```

## Design decisions

- Go single binary, no runtime deps
- Credentials hardcoded (single-user tool)
- Downloads to `cwd`, always overwrites
- `pan down --os windows` for cross-OS download edge cases
- `pan gui` starts local HTTP server + opens browser for visual download
- SDK: `github.com/aliyun/aliyun-oss-go-sdk/oss`
