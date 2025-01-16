package rpath

import (
	"fmt"
	"os"
	"path/filepath"
)

func ExecutableDir() (string, error) {
	exe_path, err := os.Executable()
	if err != nil {
		return "",
			fmt.Errorf("Can't find executable's location. Error: %w", err)
	}
	return filepath.Dir(exe_path), nil
}

// Returns path if it's absolute, converts path to
// absolute relative to the executable's directory otherwise
func Convert(exe_dir, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(exe_dir, path)
}
