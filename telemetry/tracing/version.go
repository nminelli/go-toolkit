package tracing

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	cachedVersion string
	versionOnce   sync.Once
)

// Version returns the version of the telemetry module by reading from the root VERSION file
// The version is cached after the first read for performance
func Version() string {
	versionOnce.Do(func() {
		// Try to read from the telemetry module's VERSION file
		versionPath := filepath.Join("..", "VERSION")
		if content, err := os.ReadFile(versionPath); err == nil {
			cachedVersion = strings.TrimSpace(string(content))
		} else {
			// Fallback to default version if file reading fails
			cachedVersion = "1.0.0"
		}
	})
	return cachedVersion
}
