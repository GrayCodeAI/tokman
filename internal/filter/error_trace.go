package filter

import (
	"regexp"
	"strings"
)

// ErrorTraceFilter compresses error stack traces to essential information.
// Research-based: Error Message Compression (2024) - achieves 4x compression
// by showing only: error type, first user file, line number, and message.
//
// Key insight: Stack traces are often 50+ lines, but only 2-3 lines are actionable.
// Framework internals, standard library paths, and repetitive frames can be collapsed.
type ErrorTraceFilter struct {
	// Patterns for detecting stack traces
	pythonPattern  *regexp.Regexp
	jsPattern      *regexp.Regexp
	goPattern      *regexp.Regexp
	rustPattern    *regexp.Regexp
	javaPattern    *regexp.Regexp
	errorPattern   *regexp.Regexp
	frameworkPaths []string
}

// NewErrorTraceFilter creates a new error trace compressor.
func NewErrorTraceFilter() *ErrorTraceFilter {
	return &ErrorTraceFilter{
		// Python: File "path/to/file.py", line 42, in function
		pythonPattern: regexp.MustCompile(`File "([^"]+)", line (\d+), in (\S+)`),
		// JavaScript: at functionName (file.js:42:10)
		jsPattern: regexp.MustCompile(`at\s+(\S+)\s+\(([^:]+):(\d+):\d+\)`),
		// Go: path/to/file.go:42 +0xabc
		goPattern: regexp.MustCompile(`([^\s]+\.go):(\d+)(?:\s+\+0x[0-9a-f]+)?`),
		// Rust: at src/main.rs:42
		rustPattern: regexp.MustCompile(`at\s+([^:]+\.rs):(\d+)`),
		// Java: at com.example.Class.method(File.java:42)
		javaPattern: regexp.MustCompile(`at\s+([\w.]+)\(([\w.]+):(\d+)\)`),
		// Generic error line
		errorPattern: regexp.MustCompile(`^(Error|Exception|panic|FATAL|CRITICAL)[:\s](.+)$`),
		// Framework/library paths to collapse
		frameworkPaths: []string{
			"/usr/lib/", "/usr/local/lib/",
			"/lib/", "node_modules/",
			".cargo/registry/", ".rustup/",
			"/go/src/", "/go/pkg/mod/",
			"site-packages/", "__pycache__/",
			"node_modules/", ".npm/",
			"/usr/share/", "/opt/homebrew/",
		},
	}
}

// Name returns the filter name.
func (f *ErrorTraceFilter) Name() string {
	return "error_trace"
}

// Apply compresses error traces in the output.
func (f *ErrorTraceFilter) Apply(input string, mode Mode) (string, int) {
	original := len(input)

	// Check if this looks like error output
	if !f.isErrorOutput(input) {
		return input, 0
	}

	// Detect language from error format
	lang := f.detectLanguage(input)

	// Compress based on language
	output := f.compressTrace(input, lang, mode)

	bytesSaved := original - len(output)
	tokensSaved := bytesSaved / 4

	return output, tokensSaved
}

// isErrorOutput checks if the input contains error/trace patterns
func (f *ErrorTraceFilter) isErrorOutput(input string) bool {
	indicators := []string{
		"Error:", "error:", "ERROR:",
		"Exception", "Traceback",
		"panic:", "PANIC",
		"FATAL", "Fatal", "fatal",
		"Stack trace", "stack trace",
		"at ", // JS/Java stack frames
	}

	for _, ind := range indicators {
		if strings.Contains(input, ind) {
			return true
		}
	}

	return false
}

// detectLanguage determines the programming language from error format
func (f *ErrorTraceFilter) detectLanguage(input string) string {
	if strings.Contains(input, "Traceback") || f.pythonPattern.MatchString(input) {
		return "python"
	}
	if strings.Contains(input, "panic:") || f.goPattern.MatchString(input) {
		return "go"
	}
	if f.jsPattern.MatchString(input) {
		return "javascript"
	}
	if f.rustPattern.MatchString(input) {
		return "rust"
	}
	if f.javaPattern.MatchString(input) {
		return "java"
	}
	return "unknown"
}

// compressTrace compresses stack traces for a specific language
func (f *ErrorTraceFilter) compressTrace(input, lang string, mode Mode) string {
	switch lang {
	case "python":
		return f.compressPythonTrace(input, mode)
	case "go":
		return f.compressGoTrace(input, mode)
	case "javascript":
		return f.compressJSTrace(input, mode)
	case "rust":
		return f.compressRustTrace(input, mode)
	default:
		return f.compressGenericTrace(input, mode)
	}
}

// compressPythonTrace compresses Python stack traces
func (f *ErrorTraceFilter) compressPythonTrace(input string, mode Mode) string {
	lines := strings.Split(input, "\n")
	var result []string
	var stackFrames []string
	inTraceback := false
	errorType := ""
	errorMsg := ""
	userFrames := []stackFrame{}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect traceback start
		if strings.HasPrefix(trimmed, "Traceback (most recent call last)") {
			inTraceback = true
			continue
		}

		// Parse stack frames
		if inTraceback {
			if matches := f.pythonPattern.FindStringSubmatch(line); len(matches) == 4 {
				file := matches[1]
				lineNum := matches[2]
				funcName := matches[3]

				isFramework := f.isFrameworkPath(file)
				frame := stackFrame{
					file:        file,
					line:        lineNum,
					function:    funcName,
					isFramework: isFramework,
				}

				if !isFramework {
					userFrames = append(userFrames, frame)
				}
				stackFrames = append(stackFrames, line)
				continue
			}

			// Error line (ends traceback)
			if idx := strings.Index(trimmed, ": "); idx > 0 {
				errorType = trimmed[:idx]
				errorMsg = trimmed[idx+2:]
				inTraceback = false
			}
		} else {
			// Not in traceback - keep the line
			result = append(result, line)
		}
	}

	// Build compressed output
	if errorType != "" {
		result = append(result, "")
		result = append(result, f.formatCompressedError(errorType, errorMsg, userFrames, len(stackFrames)))
	}

	return strings.Join(result, "\n")
}

// compressGoTrace compresses Go panic/error traces
func (f *ErrorTraceFilter) compressGoTrace(input string, mode Mode) string {
	lines := strings.Split(input, "\n")
	var result []string
	var stackFrames []stackFrame
	panicMsg := ""
	inPanic := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect panic
		if strings.HasPrefix(trimmed, "panic:") {
			inPanic = true
			panicMsg = strings.TrimPrefix(trimmed, "panic:")
			panicMsg = strings.TrimSpace(panicMsg)
			continue
		}

		// Parse Go stack frames
		if inPanic || strings.HasPrefix(trimmed, "goroutine") {
			if matches := f.goPattern.FindStringSubmatch(line); len(matches) >= 3 {
				file := matches[1]
				lineNum := matches[2]

				isFramework := f.isFrameworkPath(file)
				frame := stackFrame{
					file:        file,
					line:        lineNum,
					isFramework: isFramework,
					raw:         line,
				}

				stackFrames = append(stackFrames, frame)
			}
			continue
		}

		result = append(result, line)
	}

	// Build compressed output
	if panicMsg != "" && len(stackFrames) > 0 {
		userFrames := f.filterUserFrames(stackFrames)
		result = append(result, "")
		result = append(result, f.formatGoPanic(panicMsg, userFrames, len(stackFrames)))
	}

	return strings.Join(result, "\n")
}

// compressJSTrace compresses JavaScript stack traces
func (f *ErrorTraceFilter) compressJSTrace(input string, mode Mode) string {
	lines := strings.Split(input, "\n")
	var result []string
	var stackFrames []stackFrame
	errorType := ""
	errorMsg := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Parse JS stack frames: at functionName (file.js:42:10)
		if matches := f.jsPattern.FindStringSubmatch(line); len(matches) == 4 {
			funcName := matches[1]
			file := matches[2]
			lineNum := matches[3]

			isFramework := f.isFrameworkPath(file)
			frame := stackFrame{
				file:        file,
				line:        lineNum,
				function:    funcName,
				isFramework: isFramework,
			}

			stackFrames = append(stackFrames, frame)
			continue
		}

		// Error type line
		if idx := strings.Index(trimmed, ": "); idx > 0 && !strings.HasPrefix(trimmed, "at ") {
			if errorType == "" {
				errorType = trimmed[:idx]
				errorMsg = trimmed[idx+2:]
			}
			continue
		}

		// Keep non-stack lines
		if !strings.HasPrefix(trimmed, "at ") {
			result = append(result, line)
		}
	}

	// Build compressed output
	if errorType != "" {
		userFrames := f.filterUserFrames(stackFrames)
		result = append(result, "")
		result = append(result, f.formatCompressedError(errorType, errorMsg, userFrames, len(stackFrames)))
	}

	return strings.Join(result, "\n")
}

// compressRustTrace compresses Rust panic traces
func (f *ErrorTraceFilter) compressRustTrace(input string, mode Mode) string {
	lines := strings.Split(input, "\n")
	var result []string
	var stackFrames []stackFrame
	panicMsg := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Parse Rust stack frames: at src/main.rs:42
		if matches := f.rustPattern.FindStringSubmatch(line); len(matches) == 3 {
			file := matches[1]
			lineNum := matches[2]

			isFramework := f.isFrameworkPath(file)
			frame := stackFrame{
				file:        file,
				line:        lineNum,
				isFramework: isFramework,
			}

			stackFrames = append(stackFrames, frame)
			continue
		}

		// Panic message
		if strings.HasPrefix(trimmed, "panicked at") || strings.Contains(trimmed, "' panicked at ") {
			panicMsg = trimmed
			continue
		}

		result = append(result, line)
	}

	// Build compressed output
	if panicMsg != "" && len(stackFrames) > 0 {
		userFrames := f.filterUserFrames(stackFrames)
		result = append(result, "")
		result = append(result, f.formatRustPanic(panicMsg, userFrames, len(stackFrames)))
	}

	return strings.Join(result, "\n")
}

// compressGenericTrace generic compression for unknown formats
func (f *ErrorTraceFilter) compressGenericTrace(input string, mode Mode) string {
	lines := strings.Split(input, "\n")
	var result []string
	omitted := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this is a framework/internal line
		if f.isFrameworkPath(trimmed) {
			omitted++
			continue
		}

		// Keep important lines
		if f.isImportantLine(trimmed) {
			result = append(result, line)
		} else if omitted > 0 {
			result = append(result, "... ["+itoa(omitted)+" lines omitted]")
			omitted = 0
		}
	}

	if omitted > 0 {
		result = append(result, "... ["+itoa(omitted)+" lines omitted]")
	}

	return strings.Join(result, "\n")
}

// stackFrame represents a single stack frame
type stackFrame struct {
	file        string
	line        string
	function    string
	isFramework bool
	raw         string
}

// isFrameworkPath checks if a path is a framework/library
func (f *ErrorTraceFilter) isFrameworkPath(path string) bool {
	for _, fwPath := range f.frameworkPaths {
		if strings.Contains(path, fwPath) {
			return true
		}
	}
	return false
}

// filterUserFrames returns only non-framework frames
func (f *ErrorTraceFilter) filterUserFrames(frames []stackFrame) []stackFrame {
	var userFrames []stackFrame
	for _, frame := range frames {
		if !frame.isFramework {
			userFrames = append(userFrames, frame)
		}
	}
	return userFrames
}

// isImportantLine checks if a line contains important information
func (f *ErrorTraceFilter) isImportantLine(line string) bool {
	important := []string{
		"error:", "Error:", "ERROR:",
		"panic:", "PANIC",
		"fatal:", "Fatal:", "FATAL",
		"exception:", "Exception:",
	}

	for _, imp := range important {
		if strings.Contains(line, imp) {
			return true
		}
	}

	// Check for file:line patterns
	filePatterns := []string{".go:", ".py:", ".js:", ".ts:", ".rs:", ".java:", ".c:", ".cpp:"}
	for _, pattern := range filePatterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}

	return false
}

// formatCompressedError formats a compressed error message
func (f *ErrorTraceFilter) formatCompressedError(errorType, errorMsg string, userFrames []stackFrame, totalFrames int) string {
	var result []string

	// Main error line
	if len(userFrames) > 0 {
		frame := userFrames[0]
		result = append(result, errorType+": "+frame.file+":"+frame.line+" - "+frame.function+"()")
	} else {
		result = append(result, errorType+": (unknown location)")
	}

	// Error message
	if errorMsg != "" {
		if len(errorMsg) > 100 {
			errorMsg = errorMsg[:97] + "..."
		}
		result = append(result, "└─ "+errorMsg)
	}

	// Additional user frames (max 2 more)
	for i := 1; i < len(userFrames) && i < 3; i++ {
		frame := userFrames[i]
		result = append(result, "   at "+frame.file+":"+frame.line+" - "+frame.function+"()")
	}

	// Omitted frames count
	if totalFrames > len(userFrames) {
		omitted := totalFrames - len(userFrames)
		if omitted > 0 {
			result = append(result, "["+itoa(omitted)+" stack frames omitted]")
		}
	}

	return strings.Join(result, "\n")
}

// formatGoPanic formats a compressed Go panic
func (f *ErrorTraceFilter) formatGoPanic(msg string, userFrames []stackFrame, totalFrames int) string {
	var result []string

	// Main panic line
	if len(userFrames) > 0 {
		frame := userFrames[0]
		result = append(result, "panic: "+frame.file+":"+frame.line)
	} else {
		result = append(result, "panic: (unknown location)")
	}

	// Panic message
	if msg != "" {
		if len(msg) > 100 {
			msg = msg[:97] + "..."
		}
		result = append(result, "└─ "+msg)
	}

	// Additional user frames
	for i := 1; i < len(userFrames) && i < 3; i++ {
		frame := userFrames[i]
		result = append(result, "   at "+frame.file+":"+frame.line)
	}

	// Omitted frames
	omitted := totalFrames - len(userFrames)
	if omitted > 0 {
		result = append(result, "["+itoa(omitted)+" stack frames omitted]")
	}

	return strings.Join(result, "\n")
}

// formatRustPanic formats a compressed Rust panic
func (f *ErrorTraceFilter) formatRustPanic(msg string, userFrames []stackFrame, totalFrames int) string {
	var result []string

	// Main panic line
	if len(userFrames) > 0 {
		frame := userFrames[0]
		result = append(result, "panic at "+frame.file+":"+frame.line)
	} else {
		result = append(result, "panic: (unknown location)")
	}

	// Panic message (truncated)
	if msg != "" {
		if len(msg) > 100 {
			msg = msg[:97] + "..."
		}
		result = append(result, "└─ "+msg)
	}

	// Omitted frames
	omitted := totalFrames - len(userFrames)
	if omitted > 0 {
		result = append(result, "["+itoa(omitted)+" stack frames omitted]")
	}

	return strings.Join(result, "\n")
}
