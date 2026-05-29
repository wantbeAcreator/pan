# pan

Aliyun OSS-based file sync tool. Upload software to OSS once, download everywhere with one command.

## Usage

```bash
# Download all files for current OS to current directory
pan down

# Upload a file (auto-prefixed by OS)
pan up ./my-tool.exe

# Open browser-based download interface
pan gui

# Cross-OS: download windows files on linux
pan down --os windows
```

## Build

```bash
make          # produces dist/pan-linux-amd64 and dist/pan-windows-amd64.exe
```

## OSS Configuration

Edit `internal/oss/config.go` with your credentials before building.
