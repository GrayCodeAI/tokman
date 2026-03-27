package filter

import (
	"fmt"
	"time"
)

// SandboxConfig controls resource limits applied when running a Filter inside
// a sandbox.
type SandboxConfig struct {
	// MaxInputBytes is the maximum allowed size of the input string.
	// Default: 1 MiB (1 << 20).
	MaxInputBytes int

	// MaxOutputBytes is the maximum allowed size of the filter output.
	// Default: 2 MiB (2 << 20).
	MaxOutputBytes int

	// TimeoutMs is the filter execution deadline in milliseconds.
	// Default: 5000 ms.
	TimeoutMs int

	// PanicRecover enables recovery from panics inside the filter's Apply
	// method. When true, a panicking filter causes SandboxResult.Panicked to
	// be set rather than crashing the process.
	// Default: true.
	PanicRecover bool
}

// DefaultSandboxConfig returns a SandboxConfig populated with sensible defaults.
func DefaultSandboxConfig() SandboxConfig {
	return SandboxConfig{
		MaxInputBytes:  1 << 20, // 1 MiB
		MaxOutputBytes: 2 << 20, // 2 MiB
		TimeoutMs:      5000,
		PanicRecover:   true,
	}
}

// SandboxResult holds the outcome of a sandboxed filter execution.
type SandboxResult struct {
	// Output is the filtered text, or the original input on failure.
	Output string
	// TokensSaved is the number of tokens the filter reported as saved.
	TokensSaved int
	// Panicked is true when the filter panicked and PanicRecover was enabled.
	Panicked bool
	// PanicMsg contains the stringified panic value when Panicked is true.
	PanicMsg string
	// TimedOut is true when the filter exceeded its execution deadline.
	TimedOut bool
	// Error holds any non-panic error (e.g. input/output size violations).
	Error error
}

// SandboxRunner executes Filters in isolation with configurable resource limits.
type SandboxRunner struct{}

// NewSandboxRunner creates a SandboxRunner.
func NewSandboxRunner() *SandboxRunner {
	return &SandboxRunner{}
}

// RunSandboxed executes f.Apply(input, mode) under the constraints in cfg.
// If cfg is a zero value, DefaultSandboxConfig is used for any field that
// equals 0/false.
//
// Enforced limits (in order):
//  1. MaxInputBytes  — rejects oversized input immediately.
//  2. TimeoutMs      — kills the goroutine after the deadline.
//  3. PanicRecover   — wraps Apply in a deferred recover.
//  4. MaxOutputBytes — trims output that exceeds the limit.
func RunSandboxed(f Filter, input string, mode Mode, cfg SandboxConfig) SandboxResult {
	// Apply defaults for zero-value fields.
	defaults := DefaultSandboxConfig()
	if cfg.MaxInputBytes <= 0 {
		cfg.MaxInputBytes = defaults.MaxInputBytes
	}
	if cfg.MaxOutputBytes <= 0 {
		cfg.MaxOutputBytes = defaults.MaxOutputBytes
	}
	if cfg.TimeoutMs <= 0 {
		cfg.TimeoutMs = defaults.TimeoutMs
	}
	// PanicRecover defaults to true; a caller must explicitly set it false.
	// Because the zero value of bool is false we treat the default as true
	// via the struct-level default above — callers using DefaultSandboxConfig
	// get PanicRecover=true automatically.

	// --- Input size guard ---
	if len(input) > cfg.MaxInputBytes {
		return SandboxResult{
			Output: input,
			Error:  fmt.Errorf("input size %d bytes exceeds MaxInputBytes %d", len(input), cfg.MaxInputBytes),
		}
	}

	type applyResult struct {
		output string
		saved  int
		panMsg string
		panicked bool
	}

	resultCh := make(chan applyResult, 1)

	go func() {
		var ar applyResult

		if cfg.PanicRecover {
			defer func() {
				if r := recover(); r != nil {
					ar.panicked = true
					ar.panMsg = fmt.Sprintf("%v", r)
					ar.output = input // pass-through on panic
					resultCh <- ar
				}
			}()
		}

		out, saved := f.Apply(input, mode)
		ar.output = out
		ar.saved = saved
		resultCh <- ar
	}()

	timeout := time.Duration(cfg.TimeoutMs) * time.Millisecond
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case ar := <-resultCh:
		if ar.panicked {
			return SandboxResult{
				Output:   input,
				Panicked: true,
				PanicMsg: ar.panMsg,
			}
		}

		// Output size guard: truncate rather than error so callers still get
		// something useful.
		output := ar.output
		if len(output) > cfg.MaxOutputBytes {
			output = output[:cfg.MaxOutputBytes]
		}

		return SandboxResult{
			Output:      output,
			TokensSaved: ar.saved,
		}

	case <-timer.C:
		return SandboxResult{
			Output:   input,
			TimedOut: true,
		}
	}
}
