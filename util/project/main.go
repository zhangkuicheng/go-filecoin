package project

import (
	"path/filepath"
	"runtime"
)

// Root return the project root joined with any path fragments passed in as arguments
func Root(paths ...string) string {
	_, currentFilePath, _, _ := runtime.Caller(0)
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(currentFilePath)))

	allPaths := append([]string{projectRoot}, paths...)

	return filepath.Join(allPaths...)
}
