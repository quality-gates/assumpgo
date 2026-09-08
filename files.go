package assumpgo

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// CollectGoFiles returns the list of .go files reachable from fromPath,
// cleaned using filepath.Clean. A trailing ... path component is treated as a
// recursive pattern rooted at its containing directory. If fromPath is a
// single file it is returned; if it is a directory it is walked recursively.
func CollectGoFiles(fromPath string) ([]string, error) {
	pathToCollect := fromPath
	cleanPattern := filepath.Clean(fromPath)
	if filepath.Base(cleanPattern) == "..." {
		pathToCollect = filepath.Dir(cleanPattern)
	}

	info, err := os.Stat(pathToCollect)
	if err != nil {
		return nil, err
	}

	cleanPath := filepath.Clean(pathToCollect)
	if !info.IsDir() {
		return []string{cleanPath}, nil
	}

	var paths []string
	err = filepath.WalkDir(cleanPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			paths = append(paths, filepath.Clean(path))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

// CollectFromList expands a comma separated list of files/directories into a
// deduplicated list of cleaned .go files. Used for the --exclude flag.
// Non-existent paths are ignored.
func CollectFromList(list string) ([]string, error) {
	var paths []string
	seen := make(map[string]bool)
	for _, item := range strings.Split(list, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		found, err := CollectGoFiles(item)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		for _, p := range found {
			clean := filepath.Clean(p)
			if !seen[clean] {
				seen[clean] = true
				paths = append(paths, clean)
			}
		}
	}

	return paths, nil
}

// CollectTargets expands a slice of target paths into a deduplicated list
// of .go files.
func CollectTargets(targets []string) ([]string, error) {
	var paths []string
	seen := make(map[string]bool)
	for _, target := range targets {
		found, err := CollectGoFiles(target)
		if err != nil {
			return nil, err
		}
		for _, p := range found {
			clean := filepath.Clean(p)
			if !seen[clean] {
				seen[clean] = true
				paths = append(paths, clean)
			}
		}
	}

	return paths, nil
}

// ContainsPath reports whether path matches any of paths, comparing absolute
// forms so relative and absolute spellings of the same file both match.
func ContainsPath(paths []string, path string) bool {
	target := identityPath(path)
	for _, p := range paths {
		if identityPath(p) == target {
			return true
		}
	}

	return false
}
