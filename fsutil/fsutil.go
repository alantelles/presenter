// Package fsutil holds the small file-system helpers shared by the main
// package and the bible package, so both stop keeping their own copies of
// the same os.Stat/os.MkdirAll/os.WriteFile/os.ReadFile boilerplate.
package fsutil

import "os"

// PathExists reports whether path exists, distinguishing "does not exist"
// from other stat errors (e.g. permission issues).
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// CreateFolder ensures path exists, creating it (and any missing parents)
// if needed. It reports whether the folder was actually created, so
// callers can decide whether that's worth logging.
func CreateFolder(path string) (bool, error) {
	exists, err := PathExists(path)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return false, err
	}
	return true, nil
}

// WriteTextFile writes content to path, overwriting it if it already exists.
func WriteTextFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// ReadTextFile reads the whole content of path.
func ReadTextFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
