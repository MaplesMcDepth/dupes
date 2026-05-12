package main

import (
	"crypto/sha256"
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

Examples:
  dupes /path                  # Find duplicates
  dupes -r /path               # Recursive
  dupes -d /path               # Delete duplicates
  dupes -dn /path              # Dry run delete
  dupes -r -m 1024 /path       # Only files > 1KB
`)
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
	)
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() < 1 {
		usage()
		os.Exit(1)
	}

	root := flag.Arg(0)
	
	// Collect files
	var files []fileInfo
	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
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
		fmt.Println("No files found")
		return
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

	if len(dupes) == 0 {
		fmt.Println("No duplicates found")
		return
	}

	// Output results
	totalDupes := 0
	totalBytes := int64(0)

	for _, group := range dupes {
		fmt.Printf("\n%s (%d bytes)\n", filepath.Base(group[0].path), group[0].size)
		for i, f := range group {
			marker := "  [original]"
			if i > 0 {
				marker = "  [duplicate]"
				totalDupes++
				totalBytes += f.size
			}
			fmt.Printf("  %s%s\n", f.path, marker)
		}
	}

	fmt.Printf("\nFound %d duplicate file(s), %s wasted\n", 
		totalDupes, humanSize(totalBytes))

	// Handle deletion
	if *delete || *dryRun {
		action := "Deleting"
		if *dryRun {
			action = "Would delete"
		}
		fmt.Printf("\n%s %d duplicate file(s)...\n", action, totalDupes)
		
		deleted := 0
		for _, group := range dupes {
			for i := 1; i < len(group); i++ {
				if !*dryRun {
					if err := os.Remove(group[i].path); err != nil {
						fmt.Fprintf(os.Stderr, "  Error deleting %s: %v\n", group[i].path, err)
						continue
					}
				}
				fmt.Printf("  %s\n", group[i].path)
				deleted++
			}
		}
		fmt.Printf("\n%s %d file(s)\n", action, deleted)
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
