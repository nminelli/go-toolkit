package webapp

import (
	"bytes"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/pkg/errors"
)

// PanicInfo represents structured information about a recovered panic.
type PanicInfo struct {
	Value      interface{} // The panic value
	Message    string      // Formatted panic message
	StackTrace string      // Pretty-printed stack trace
	Error      error       // Error representation of the panic with embedded stack trace
}

// panicError wraps an error with our custom stack trace information
type panicError struct {
	cause      error
	message    string
	stackTrace string
}

func (e *panicError) Error() string {
	return e.message
}

func (e *panicError) Cause() error {
	return e.cause
}

func (e *panicError) Unwrap() error {
	return e.cause
}

// StackTrace returns the embedded stack trace as a string
func (e *panicError) StackTrace() string {
	return e.stackTrace
}

// FormatPanic takes a panic value (from recover()) and formats it into structured information.
// It returns nil if the panic value is nil.
// This function is safe to call and will not panic itself.
//
// Usage:
//
//	defer func() {
//	    if r := recover(); r != nil {
//	        panicInfo := webapp.FormatPanic(r)
//	        // Use panicInfo for logging, tracing, etc.
//	    }
//	}()
func FormatPanic(panicValue interface{}) *PanicInfo {
	defer func() {
		// Ensure this function never panics
		if r := recover(); r != nil {
			// If something goes wrong in our panic handler, we can't do much
			// but we shouldn't panic again
		}
	}()

	if panicValue == nil {
		return nil
	}

	// Safely capture stack trace first
	stackTrace := safeStackTrace()

	var message string
	var baseErr error
	var panicErr error

	// Handle different panic value types
	switch v := panicValue.(type) {
	case error:
		message = fmt.Sprintf("panic: %v", v)
		baseErr = v
	case string:
		message = fmt.Sprintf("panic: %s", v)
		baseErr = errors.New(v)
	default:
		message = fmt.Sprintf("panic: %v", v)
		baseErr = errors.New(fmt.Sprintf("%v", v))
	}

	// Create our custom error with embedded stack trace
	panicErr = &panicError{
		cause:      baseErr,
		message:    message,
		stackTrace: stackTrace,
	}

	return &PanicInfo{
		Value:      panicValue,
		Message:    message,
		StackTrace: stackTrace,
		Error:      panicErr,
	}
}

// safeStackTrace safely captures and formats the stack trace.
// It ensures no panic occurs during stack trace generation.
func safeStackTrace() string {
	defer func() {
		// Ensure stack trace capture never panics
		if r := recover(); r != nil {
			// If we can't get stack trace, return empty string
		}
	}()

	stack := debug.Stack()
	if stack == nil {
		return "stack trace unavailable"
	}

	return formatStackTrace(stack)
}

// formatStackTrace formats the raw stack trace to show only the essential user code.
// It aggressively filters out middleware and runtime noise to focus on where the panic actually occurred.
func formatStackTrace(stack []byte) string {
	defer func() {
		// Ensure formatting never panics
		if r := recover(); r != nil {
			// If formatting fails, return raw stack
		}
	}()

	if len(stack) == 0 {
		return "empty stack trace"
	}

	lines := bytes.Split(stack, []byte("\n"))
	var result []string

	// Look for the most important frames: user code where the panic occurred
	// Stack traces have function lines followed by location lines, but not always in pairs from index 0
	for i := 0; i < len(lines); i++ {
		functionLine := strings.TrimSpace(string(lines[i]))

		if functionLine == "" {
			continue
		}

		// Skip if this doesn't look like a function call
		if !strings.Contains(functionLine, "(") {
			continue
		}

		// Find the location line (next non-empty line without parentheses)
		var locationLine string
		for j := i + 1; j < len(lines); j++ {
			candidate := strings.TrimSpace(string(lines[j]))
			if candidate != "" && !strings.Contains(candidate, "(") {
				locationLine = candidate
				break
			}
		}

		// Check if this is user code we want to highlight
		if isImportantUserCode(functionLine, locationLine) {
			result = append(result, functionLine)
			if locationLine != "" {
				result = append(result, "  "+locationLine)
			}
		}
	}

	// If we found user code, return just that (maximum 2 frames to keep it clean)
	if len(result) > 0 {
		// Limit to maximum 4 lines (2 function+location pairs) for extreme cleanliness
		if len(result) > 4 {
			result = result[:4]
		}
		return strings.Join(result, "\n")
	}

	// Fallback: show ONLY the most relevant frame, not the verbose context
	return formatMinimalStackTrace(lines)
}

// isImportantUserCode determines if a stack frame represents the user code where the panic occurred.
// This function is very aggressive about filtering out framework and middleware noise.
func isImportantUserCode(functionLine, locationLine string) bool {
	// Combine both lines for analysis
	combined := functionLine + " " + locationLine

	// Primary indicators: these are almost certainly user code we want to show
	primaryIndicators := []string{
		"main.",
		"/main.go",
		"/cmd/",
		"/internal/",
		"/api/",
		"/handler",
		"/service",
		"/controller",
	}

	for _, indicator := range primaryIndicators {
		if strings.Contains(combined, indicator) {
			return true
		}
	}

	// Aggressive filtering: skip ALL framework/middleware code
	skipPatterns := []string{
		// Runtime and system
		"runtime/",
		"net/http",

		// Common frameworks and libraries
		"github.com/go-chi/chi",
		"go-toolkit/webapp",
		"go-toolkit/httprouter",

		// Middleware patterns
		"middleware/",
		"telemetryMiddleware",
		"logMiddleware",
		"headerForwarder",
		"wrapHandler",
		"ServeHTTP",
		"routeHTTP",
		"compress.go",

		// Internal patterns
		".gvm/",
		"gopanic",
		"debug.Stack",
		"FormatPanic",
		"RecoverAndFormat",

		// Function-specific middleware signatures
		".func1",
		".func2",
		".func3",
		".func4",
		".func5",
		"(*Mux)",
		"(*Router)",
		"(*Compressor)",
		"HandlerFunc",
	}

	for _, pattern := range skipPatterns {
		if strings.Contains(combined, pattern) {
			return false
		}
	}

	// If we get here, it might be user code - check if it has a recognizable file path
	// But be more restrictive: only include if it looks like a real user project path
	if strings.Contains(locationLine, ".go") &&
		!strings.Contains(locationLine, ".gvm/") &&
		!strings.Contains(locationLine, "/src/") &&
		(strings.Contains(locationLine, "/Users/") || strings.Contains(locationLine, "/home/") || strings.Contains(locationLine, "/app/")) {
		return true
	}

	return false
}

// formatMinimalStackTrace provides a very minimal fallback that shows only the most relevant frame
func formatMinimalStackTrace(lines [][]byte) string {
	var foundFrames []string

	// Find the first few non-internal frames
	for i := 0; i < len(lines)-1; i += 2 {
		functionLine := strings.TrimSpace(string(lines[i]))
		var locationLine string
		if i+1 < len(lines) {
			locationLine = strings.TrimSpace(string(lines[i+1]))
		}

		if functionLine == "" {
			continue
		}

		// Skip if this doesn't look like a function call
		if !strings.Contains(functionLine, "(") {
			continue
		}

		// Skip our own panic handling functions - be more specific
		if strings.Contains(functionLine, "webapp.FormatPanic") ||
			strings.Contains(functionLine, "webapp.safeStackTrace") ||
			strings.Contains(functionLine, "webapp.formatStackTrace") ||
			strings.Contains(functionLine, "runtime/panic.go") ||
			strings.Contains(functionLine, "runtime/debug.Stack") ||
			strings.Contains(functionLine, "runtime.") {
			continue
		}

		// Add this frame
		if locationLine != "" {
			foundFrames = append(foundFrames, functionLine)
			foundFrames = append(foundFrames, "  "+locationLine)
		}

		// Stop after finding the first relevant frame to keep it minimal
		if len(foundFrames) >= 2 {
			break
		}
	}

	if len(foundFrames) > 0 {
		return strings.Join(foundFrames, "\n")
	}

	return "panic occurred (location unavailable)"
}
