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
