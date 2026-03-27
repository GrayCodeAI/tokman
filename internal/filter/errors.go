package filter

import "fmt"

// ErrFilterFailed wraps a filter application error with context.
type ErrFilterFailed struct {
	FilterName string
	Cause      error
}

func (e *ErrFilterFailed) Error() string {
	return fmt.Sprintf("filter %q failed: %v", e.FilterName, e.Cause)
}

func (e *ErrFilterFailed) Unwrap() error {
	return e.Cause
}

// ErrBudgetExceeded is returned when output exceeds the token budget.
type ErrBudgetExceeded struct {
	Budget int
	Actual int
}

func (e *ErrBudgetExceeded) Error() string {
	return fmt.Sprintf("token budget exceeded: budget=%d actual=%d (over by %d)", e.Budget, e.Actual, e.Actual-e.Budget)
}

// ErrContentTooLarge is returned when input exceeds the maximum processable size.
type ErrContentTooLarge struct {
	MaxBytes int
	Actual   int
}

func (e *ErrContentTooLarge) Error() string {
	return fmt.Sprintf("content too large: max=%d bytes actual=%d bytes (over by %d)", e.MaxBytes, e.Actual, e.Actual-e.MaxBytes)
}

// ErrInvalidMode is returned when an unrecognized mode is used.
type ErrInvalidMode struct {
	Mode string
}

func (e *ErrInvalidMode) Error() string {
	return fmt.Sprintf("invalid filter mode %q: must be one of none, minimal, aggressive", e.Mode)
}
