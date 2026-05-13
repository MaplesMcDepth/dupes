# dupes

![CI](https://github.com/MaplesMcDepth/dupes/actions/workflows/ci.yml/badge.svg)
![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)


Find and optionally remove duplicate files.

## Install

```bash
go install github.com/MaplesMcDepth/dupes/cmd/dupes@latest
```

## Commands

```bash
dupes /path                  # Find duplicates
dupes -r /path               # Recursive search
dupes -d /path               # Delete duplicates
dupes -dn /path              # Dry run delete
dupes -r -m 1024 /path       # Only files > 1KB
```

## Options

| Flag | Description |
|------|-------------|
| `-d` | Delete duplicates (keep first) |
| `-n` | Dry run |
| `-r` | Recursive |
| `-m int` | Min file size in bytes |

## AI Agent Features

### JSON Output (`-j`)
All tools support structured JSON output for programmatic consumption:

```bash
git-standup -j              # Machine-readable commit history
dupes -j /path              # Structured duplicate report
watch -j '*.go' go test     # JSON events with output + exit codes
```

### Quiet Mode (`-q`)
Suppress human-readable output. Useful in automated workflows:

```bash
git-standup -jq             # JSON only, no headers
dupes -jq /path             # JSON only, no progress
```

### Environment Variables
- `STANDUP_DAYS` — Default days back for git-standup

### Webhook Support (watch)
POST events to a URL when files change:

```bash
watch -w http://localhost:8080/hook '*.go' go build
```

### Exit Codes
- `0` — Success / no issues found
- `1` — Error or duplicates found (dupes)
