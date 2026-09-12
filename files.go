package assumpgo

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// CollectGoFiles returns the list of .go files reachable from fromPath,
// cleaned using filepath.Clean. A trailing ... path component is treated as a
// recursive pattern rooted at its containing directory. If fromPath is a
// single file it is returned; if it is a directory it is walked recursively,
// following directory symlinks without revisiting a directory.
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
	err = walkGoFiles(cleanPath, nil, &paths)
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func walkGoFiles(path string, visited []os.FileInfo, paths *[]string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}

	for _, seen := range visited {
		if os.SameFile(seen, info) {
			return nil
		}
	}
	visited = append(visited, info)

	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		child := filepath.Join(path, entry.Name())
		childInfo, err := os.Stat(child)
		if err != nil {
			if entry.Type()&os.ModeSymlink != 0 {
				continue
			}
			return err
		}
		if childInfo.IsDir() {
			if err := walkGoFiles(child, visited, paths); err != nil {
				return err
			}
			continue
		}
		if strings.HasSuffix(child, ".go") {
			*paths = append(*paths, filepath.Clean(child))
		}
	}

	return nil
}

// CollectFromList expands a comma separated list of files/directories into a
// deduplicated list of cleaned .go files. Used for the --exclude flag.
// Non-existent paths are ignored.
func CollectFromList(list string) ([]string, error) {
	var paths []string
	var infos []os.FileInfo
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
			identity := identityPath(clean)
			if seen[identity] {
				continue
			}

			info, err := os.Stat(clean)
			if err != nil {
				seen[identity] = true
				paths = append(paths, clean)
				continue
			}

			if containsSameFile(infos, info) {
				seen[identity] = true
				continue
			}

			seen[identity] = true
			paths = append(paths, clean)
			infos = append(infos, info)
		}
	}

	return paths, nil
}

// CollectTargets expands a slice of target paths into a deduplicated list
// of .go files. Deduplication compares absolute forms and filesystem identity,
// so relative, absolute, symlink, and hard-link spellings of the same file
// collapse to one entry; the first spelling given is the one kept.
func CollectTargets(targets []string) ([]string, error) {
	var paths []string
	var infos []os.FileInfo
	seen := make(map[string]bool)
	for _, target := range targets {
		found, err := CollectGoFiles(target)
		if err != nil {
			return nil, err
		}
		for _, p := range found {
			clean := filepath.Clean(p)
			identity := identityPath(clean)
			if seen[identity] {
				continue
			}

			info, err := os.Stat(clean)
			if err != nil {
				seen[identity] = true
				paths = append(paths, clean)
				continue
			}

			if containsSameFile(infos, info) {
				seen[identity] = true
				continue
			}

			seen[identity] = true
			paths = append(paths, clean)
			infos = append(infos, info)
		}
	}

	return paths, nil
}

func containsSameFile(infos []os.FileInfo, info os.FileInfo) bool {
	for _, seen := range infos {
		if os.SameFile(seen, info) {
			return true
		}
	}
	return false
}

// ContainsPath reports whether path matches any of paths, comparing absolute
// forms and filesystem identity so path aliases of the same file both match.
func ContainsPath(paths []string, path string) bool {
	target := identityPath(path)
	for _, p := range paths {
		if identityPath(p) == target || sameFile(p, path) {
			return true
		}
	}

	return false
}

func sameFile(first, second string) bool {
	firstInfo, err := os.Stat(first)
	if err != nil {
		return false
	}
	secondInfo, err := os.Stat(second)
	if err != nil {
		return false
	}
	return os.SameFile(firstInfo, secondInfo)
}
