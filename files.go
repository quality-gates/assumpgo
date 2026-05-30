package assumpgo

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// CollectGoFiles returns the list of .go files reachable from fromPath. If
// fromPath is a single file it is returned as-is; if it is a directory it is
// walked recursively.
func CollectGoFiles(fromPath string) ([]string, error) {
	info, err := os.Stat(fromPath)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return []string{fromPath}, nil
	}

	var paths []string
	err = filepath.WalkDir(fromPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

// CollectFromList expands a comma separated list of files/directories into a
// flat list of .go files. Used for the --exclude flag.
func CollectFromList(list string) ([]string, error) {
	var paths []string
	if strings.TrimSpace(list) == "" {
		return paths, nil
	}

	for _, item := range strings.Split(list, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		found, err := CollectGoFiles(item)
		if err != nil {
			return nil, err
		}
		paths = append(paths, found...)
	}

	return paths, nil
}
