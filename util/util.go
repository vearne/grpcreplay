package util

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ListFilesRecursively traverses the directory tree rooted at dir and adds all .proto file paths to the provided StringSet.
// Returns an error if the StringSet is nil, if the directory cannot be opened, or if a read or traversal error occurs.
func ListFilesRecursively(dir string, set *StringSet) error {
	if set == nil {
		return errors.New("StringSet cannot be nil")
	}
	dir = filepath.Clean(dir)
	d, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("can't open %s: %v", dir, err)
	}
	defer d.Close()

	entries, err := d.ReadDir(0) // 0 means read all entries
	if err != nil {
		return fmt.Errorf("read %s, error: %v", dir, err)
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())

		if entry.IsDir() {
			if err := ListFilesRecursively(path, set); err != nil {
				return err
			}
		} else {
			if strings.HasSuffix(path, ".proto") {
				set.Add(path)
			}
		}
	}
	return nil
}
