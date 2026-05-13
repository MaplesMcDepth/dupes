package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

func usage() {
	fmt.Print(`dupes — Find duplicate files

Usage: dupes [options] <path>

Options:
  -d             Delete duplicates (keep first occurrence)
  -n             Dry run (show what would be deleted)
  -r             Recursive search
  -m int         Min file size in bytes (default 1)
  -j             JSON output (agent-friendly)
  -q             Quiet mode (no progress)

Examples:
  dupes /path                  # Find duplicates
  dupes -r /path               # Recursive
  dupes -d /path               # Delete duplicates
  dupes -j /path               # JSON output for agents
  dupes -jq /path              # JSON, quiet
`)
}

type FileEntry struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
	Hash string `json:"hash"`
}

type DuplicateGroup struct {
	Hash      string      `json:"hash"`
	Size      int64       `json:"size"`
	Files     []FileEntry `json:"files"`
	Original  string      `json:"original"`
}

type DupesReport struct {
	Scanned     int              `json:"scanned"`
	Duplicates  int              `json:"duplicates"`
	Groups      int              `json:"groups"`
	WastedBytes int64            `json:"wasted_bytes"`
	GroupsList  []DuplicateGroup `json:"groups_list,omitempty"`
}

type fileInfo struct {
	path string
	size int64
	hash string
}

func main() {
	var (
		delete  = flag.Bool("d", false, "Delete duplicates")
		dryRun  = flag.Bool("n", false, "Dry run")
		recursive = flag.Bool("r", false, "Recursive search")
		minSize = flag.Int64("m", 1, "Min file size in bytes")
		jsonOut = flag.Bool("j", false, "JSON output")
		quiet   = flag.Bool("q", false, "Quiet mode")
	)
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() < 1 {
		usage()
		os.Exit(1)
	}

	root := flag.Arg(0)
	
	var files []fileInfo
	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if !*recursive && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Size() < *minSize {
			return nil
		}
		files = append(files, fileInfo{path: path, size: info.Size()})
		return nil
	}

	if err := filepath.Walk(root, walkFn); err != nil {
		fmt.Fprintf(os.Stderr, "Error walking path: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		if !*quiet && !*jsonOut {
			fmt.Println("No files found")
		}
		os.Exit(0)
	}

	// Group by size first (fast check)
	sizeGroups := make(map[int64][]fileInfo)
	for _, f := range files {
		sizeGroups[f.size] = append(sizeGroups[f.size], f)
	}

	// For groups with same size, hash and find duplicates
	var dupes [][]fileInfo
	for _, group := range sizeGroups {
		if len(group) < 2 {
			continue
		}
		hashGroups := make(map[string][]fileInfo)
		for _, f := range group {
			hash, err := hashFile(f.path)
			if err != nil {
				continue
			}
			f.hash = hash
			hashGroups[hash] = append(hashGroups[hash], f)
		}
		for _, hg := range hashGroups {
			if len(hg) > 1 {
				sort.Slice(hg, func(i, j int) bool {
					return hg[i].path < hg[j].path
				})
				dupes = append(dupes, hg)
			}
		}
	}

	// Build report
	totalDupes := 0
	totalBytes := int64(0)
	var groupsList []DuplicateGroup

	for _, group := range dupes {
		var entries []FileEntry
		for _, f := range group {
			entries = append(entries, FileEntry{
				Path: f.path,
				Size: f.size,
				Hash: f.hash,
			})
		}
		groupsList = append(groupsList, DuplicateGroup{
			Hash:     group[0].hash,
			Size:     group[0].size,
			Files:    entries,
			Original: group[0].path,
		})
		for i := 1; i < len(group); i++ {
			totalDupes++
			totalBytes += group[i].size
		}
	}

	if *jsonOut {
		report := DupesReport{
			Scanned:     len(files),
			Duplicates:  totalDupes,
			Groups:      len(dupes),
			WastedBytes: totalBytes,
			GroupsList:  groupsList,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(report)
		return
	}

	if len(dupes) == 0 {
		if !*quiet {
			fmt.Println("No duplicates found")
		}
		os.Exit(0)
	}

	for _, group := range dupes {
		if !*quiet {
			fmt.Printf("\n%s (%d bytes)\n", filepath.Base(group[0].path), group[0].size)
		}
		for i, f := range group {
			marker := "  [original]"
			if i > 0 {
				marker = "  [duplicate]"
			}
			if !*quiet {
				fmt.Printf("  %s%s\n", f.path, marker)
			}
		}
	}

	if !*quiet {
		fmt.Printf("\nFound %d duplicate file(s), %s wasted\n", 
			totalDupes, humanSize(totalBytes))
	}

	// Handle deletion
	if *delete || *dryRun {
		action := "Deleting"
		if *dryRun {
			action = "Would delete"
		}
		if !*quiet {
			fmt.Printf("\n%s %d duplicate file(s)...\n", action, totalDupes)
		}
		
		deleted := 0
		for _, group := range dupes {
			for i := 1; i < len(group); i++ {
				if !*dryRun {
					if err := os.Remove(group[i].path); err != nil {
						if !*quiet {
							fmt.Fprintf(os.Stderr, "  Error deleting %s: %v\n", group[i].path, err)
						}
						continue
					}
				}
				if !*quiet {
					fmt.Printf("  %s\n", group[i].path)
				}
				deleted++
			}
		}
		if !*quiet {
			fmt.Printf("\n%s %d file(s)\n", action, deleted)
		}
	}
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func humanSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
