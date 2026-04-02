// Package undo provides undo/redo functionality for CLI operations
package undo

import (
	"fmt"
	"sync"
	"time"
)

// Action represents an undoable action
type Action struct {
	ID          string
	Command     string
	Args        []string
	Description string
	Timestamp   time.Time
	User        string
	State       map[string]interface{}
	Rollback    func() error
}

// UndoManager manages undo/redo operations
type UndoManager struct {
	undoStack  []Action
	redoStack  []Action
	maxHistory int
	mu         sync.RWMutex
}

// NewUndoManager creates a new undo manager
func NewUndoManager(maxHistory int) *UndoManager {
	if maxHistory <= 0 {
		maxHistory = 100
	}

	return &UndoManager{
		undoStack:  make([]Action, 0, maxHistory),
		redoStack:  make([]Action, 0, maxHistory),
		maxHistory: maxHistory,
	}
}

// Record records an action for undo
func (um *UndoManager) Record(action Action) {
	um.mu.Lock()
	defer um.mu.Unlock()

	if action.ID == "" {
		action.ID = fmt.Sprintf("action-%d", time.Now().UnixNano())
	}
	if action.Timestamp.IsZero() {
		action.Timestamp = time.Now()
	}

	um.undoStack = append(um.undoStack, action)

	// Clear redo stack when new action is recorded
	um.redoStack = make([]Action, 0)

	// Trim history if needed
	if len(um.undoStack) > um.maxHistory {
		um.undoStack = um.undoStack[len(um.undoStack)-um.maxHistory:]
	}
}

// Undo undoes the last action
func (um *UndoManager) Undo() (*Action, error) {
	um.mu.Lock()
	defer um.mu.Unlock()

	if len(um.undoStack) == 0 {
		return nil, fmt.Errorf("nothing to undo")
	}

	// Get last action
	action := um.undoStack[len(um.undoStack)-1]
	um.undoStack = um.undoStack[:len(um.undoStack)-1]

	// Execute rollback if available
	if action.Rollback != nil {
		if err := action.Rollback(); err != nil {
			return nil, fmt.Errorf("rollback failed: %w", err)
		}
	}

	// Add to redo stack
	um.redoStack = append(um.redoStack, action)

	return &action, nil
}

// Redo redoes the last undone action
func (um *UndoManager) Redo() (*Action, error) {
	um.mu.Lock()
	defer um.mu.Unlock()

	if len(um.redoStack) == 0 {
		return nil, fmt.Errorf("nothing to redo")
	}

	// Get last redo action
	action := um.redoStack[len(um.redoStack)-1]
	um.redoStack = um.redoStack[:len(um.redoStack)-1]

	// Add back to undo stack
	um.undoStack = append(um.undoStack, action)

	return &action, nil
}

// CanUndo checks if undo is possible
func (um *UndoManager) CanUndo() bool {
	um.mu.RLock()
	defer um.mu.RUnlock()
	return len(um.undoStack) > 0
}

// CanRedo checks if redo is possible
func (um *UndoManager) CanRedo() bool {
	um.mu.RLock()
	defer um.mu.RUnlock()
	return len(um.redoStack) > 0
}

// GetHistory returns the undo history
func (um *UndoManager) GetHistory() []Action {
	um.mu.RLock()
	defer um.mu.RUnlock()

	history := make([]Action, len(um.undoStack))
	copy(history, um.undoStack)
	return history
}

// Clear clears all history
func (um *UndoManager) Clear() {
	um.mu.Lock()
	defer um.mu.Unlock()

	um.undoStack = make([]Action, 0)
	um.redoStack = make([]Action, 0)
}

// UndoReport generates a report of undo history
type UndoReport struct {
	TotalActions int
	OldestAction time.Time
	NewestAction time.Time
	Actions      []ActionSummary
}

// ActionSummary represents a summary of an action
type ActionSummary struct {
	ID          string
	Command     string
	Description string
	Timestamp   time.Time
	User        string
}

// GenerateReport generates an undo report
func (um *UndoManager) GenerateReport() *UndoReport {
	um.mu.RLock()
	defer um.mu.RUnlock()

	report := &UndoReport{
		TotalActions: len(um.undoStack),
		Actions:      make([]ActionSummary, 0, len(um.undoStack)),
	}

	for _, action := range um.undoStack {
		if report.OldestAction.IsZero() || action.Timestamp.Before(report.OldestAction) {
			report.OldestAction = action.Timestamp
		}
		if action.Timestamp.After(report.NewestAction) {
			report.NewestAction = action.Timestamp
		}

		report.Actions = append(report.Actions, ActionSummary{
			ID:          action.ID,
			Command:     action.Command,
			Description: action.Description,
			Timestamp:   action.Timestamp,
			User:        action.User,
		})
	}

	return report
}
