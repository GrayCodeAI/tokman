package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileSummary represents a compressed summary of a file's contents
type FileSummary struct {
	Path         string   `json:"path"`
	Imports      []string `json:"imports,omitempty"`
	Exports      []string `json:"exports,omitempty"`
	Types        []string `json:"types,omitempty"`
	Functions    []string `json:"functions,omitempty"`
	KeyPatterns  []string `json:"key_patterns,omitempty"`
	TokenCount   int      `json:"token_count"`
	LastModified int64    `json:"last_modified"`
}

// TypeDefinition represents a type/class/interface definition
type TypeDefinition struct {
	Name        string   `json:"name"`
	Kind        string   `json:"kind"` // struct, class, interface, enum
	File        string   `json:"file"`
	Fields      []string `json:"fields,omitempty"`
	Methods     []string `json:"methods,omitempty"`
	DocComments []string `json:"doc_comments,omitempty"`
}

// Error represents a captured error with context
type Error struct {
	Message   string `json:"message"`
	File      string `json:"file,omitempty"`
	Line      int    `json:"line,omitempty"`
	Stack     string `json:"stack,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// SessionContext holds shared context between agents
type SessionContext struct {
	mu sync.RWMutex

	ID           string `json:"id"`
	Agent        string `json:"agent"`
	Created      int64  `json:"created"`
	LastAccessed int64  `json:"last_accessed"`

	// File tracking
	FilesSeen map[string]*FileSummary `json:"files_seen"`

	// Type definitions
	TypesDefined map[string]*TypeDefinition `json:"types_defined"`

	// Error tracking
	ErrorsSeen []Error `json:"errors_seen"`

	// Shared patterns
	Patterns map[string]int `json:"patterns"` // pattern -> occurrence count

	// Commands executed
	Commands []CommandRecord `json:"commands"`

	// Key-value store for custom data
	Extras map[string]interface{} `json:"extras"`
}

// CommandRecord tracks a command execution
type CommandRecord struct {
	Command     string `json:"command"`
	ExitCode    int    `json:"exit_code"`
	TokensSaved int    `json:"tokens_saved"`
	Timestamp   int64  `json:"timestamp"`
}

// Manager manages session contexts
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*SessionContext

	// Directory for session persistence
	sessionDir string

	// Current active session
	active *SessionContext
}

// NewManager creates a new session manager
func NewManager(sessionDir string) *Manager {
	if sessionDir == "" {
		home, _ := os.UserHomeDir()
		sessionDir = filepath.Join(home, ".local", "share", "tokman", "sessions")
	}

	m := &Manager{
		sessions:   make(map[string]*SessionContext),
		sessionDir: sessionDir,
	}

	// Ensure directory exists
	os.MkdirAll(sessionDir, 0755)

	// Load existing sessions
	m.loadSessions()

	return m
}

// loadSessions loads all persisted sessions
func (m *Manager) loadSessions() {
	entries, err := os.ReadDir(m.sessionDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".json") {
			path := filepath.Join(m.sessionDir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			var ctx SessionContext
			if err := json.Unmarshal(data, &ctx); err != nil {
				continue
			}

			m.sessions[ctx.ID] = &ctx
		}
	}
}

// NewSession creates a new session context
func (m *Manager) NewSession(agent string) *SessionContext {
	m.mu.Lock()
	defer m.mu.Unlock()

	ctx := &SessionContext{
		ID:           generateSessionID(),
		Agent:        agent,
		Created:      time.Now().Unix(),
		LastAccessed: time.Now().Unix(),
		FilesSeen:    make(map[string]*FileSummary),
		TypesDefined: make(map[string]*TypeDefinition),
		ErrorsSeen:   make([]Error, 0),
		Patterns:     make(map[string]int),
		Commands:     make([]CommandRecord, 0),
		Extras:       make(map[string]interface{}),
	}

	m.sessions[ctx.ID] = ctx
	m.active = ctx

	// Persist immediately
	m.persist(ctx)

	return ctx
}

// GetSession retrieves a session by ID
func (m *Manager) GetSession(id string) (*SessionContext, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ctx, ok := m.sessions[id]
	return ctx, ok
}

// ActiveSession returns the current active session
func (m *Manager) ActiveSession() *SessionContext {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.active
}

// SetActive sets the active session
func (m *Manager) SetActive(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ctx, ok := m.sessions[id]
	if !ok {
		return fmt.Errorf("session %q not found", id)
	}

	m.active = ctx
	ctx.LastAccessed = time.Now().Unix()

	return nil
}

// ListSessions returns all sessions
func (m *Manager) ListSessions() []*SessionContext {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*SessionContext, 0, len(m.sessions))
	for _, ctx := range m.sessions {
		sessions = append(sessions, ctx)
	}
	return sessions
}

// DeleteSession removes a session
func (m *Manager) DeleteSession(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.sessions[id]; !ok {
		return fmt.Errorf("session %q not found", id)
	}

	delete(m.sessions, id)

	// Remove persisted file
	path := filepath.Join(m.sessionDir, id+".json")
	os.Remove(path)

	return nil
}

// persist saves a session to disk
func (m *Manager) persist(ctx *SessionContext) error {
	data, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(m.sessionDir, ctx.ID+".json")
	return os.WriteFile(path, data, 0644)
}

// Sync persists the session to disk
func (s *SessionContext) Sync(manager *Manager) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.LastAccessed = time.Now().Unix()
	return manager.persist(s)
}

// RecordFile adds or updates a file summary
func (s *SessionContext) RecordFile(summary *FileSummary) {
	s.mu.Lock()
	defer s.mu.Unlock()

	summary.LastModified = time.Now().Unix()
	s.FilesSeen[summary.Path] = summary
}

// RecordType adds or updates a type definition
func (s *SessionContext) RecordType(def *TypeDefinition) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TypesDefined[def.Name] = def
}

// RecordError adds an error to the session
func (s *SessionContext) RecordError(err Error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	err.Timestamp = time.Now().Unix()
	s.ErrorsSeen = append(s.ErrorsSeen, err)
}

// RecordCommand adds a command execution record
func (s *SessionContext) RecordCommand(cmd string, exitCode, tokensSaved int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Commands = append(s.Commands, CommandRecord{
		Command:     cmd,
		ExitCode:    exitCode,
		TokensSaved: tokensSaved,
		Timestamp:   time.Now().Unix(),
	})
}

// RecordPattern increments a pattern occurrence
func (s *SessionContext) RecordPattern(pattern string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Patterns[pattern]++
}

// HasFile checks if a file has been seen
func (s *SessionContext) HasFile(path string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.FilesSeen[path]
	return ok
}

// GetFile retrieves a file summary
func (s *SessionContext) GetFile(path string) (*FileSummary, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	summary, ok := s.FilesSeen[path]
	return summary, ok
}

// GetType retrieves a type definition
func (s *SessionContext) GetType(name string) (*TypeDefinition, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	def, ok := s.TypesDefined[name]
	return def, ok
}

// GetCommonPatterns returns patterns seen more than once
func (s *SessionContext) GetCommonPatterns(minOccurrences int) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var patterns []string
	for p, count := range s.Patterns {
		if count >= minOccurrences {
			patterns = append(patterns, p)
		}
	}
	return patterns
}

// GetRecentErrors returns errors from the last N minutes
func (s *SessionContext) GetRecentErrors(minutes int) []Error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cutoff := time.Now().Add(-time.Duration(minutes) * time.Minute).Unix()
	var recent []Error
	for _, err := range s.ErrorsSeen {
		if err.Timestamp >= cutoff {
			recent = append(recent, err)
		}
	}
	return recent
}

// TokenSavings returns total tokens saved in this session
func (s *SessionContext) TokenSavings() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var total int
	for _, cmd := range s.Commands {
		total += cmd.TokensSaved
	}
	return total
}

// Summary generates a text summary of the session
func (s *SessionContext) Summary() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Session: %s (Agent: %s)\n", s.ID[:8], s.Agent))
	sb.WriteString(fmt.Sprintf("Duration: %s\n", time.Since(time.Unix(s.Created, 0)).Round(time.Minute)))
	sb.WriteString(fmt.Sprintf("Tokens Saved: %d\n\n", s.TokenSavings()))

	if len(s.FilesSeen) > 0 {
		sb.WriteString(fmt.Sprintf("Files Seen: %d\n", len(s.FilesSeen)))
	}

	if len(s.TypesDefined) > 0 {
		sb.WriteString(fmt.Sprintf("Types Defined: %d\n", len(s.TypesDefined)))
	}

	if len(s.ErrorsSeen) > 0 {
		sb.WriteString(fmt.Sprintf("Errors Encountered: %d\n", len(s.ErrorsSeen)))
	}

	return sb.String()
}

// generateSessionID creates a unique session identifier
func generateSessionID() string {
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), randomString(6))
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().Nanosecond()%len(letters)]
	}
	return string(b)
}
