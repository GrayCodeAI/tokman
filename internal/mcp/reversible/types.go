// Package reversible implements Claw Compactor's Rewind engine for reversible compression.
package reversible

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ContentType represents the classification of content for compression.
type ContentType int

const (
	TypeLossless ContentType = iota // No loss acceptable (code, config)
	TypeLossy                       // Minor loss OK (logs with timestamps)
	TypeSemantic                    // Semantic preservation only (docs)
)

// Entry represents a stored compressed entry.
type Entry struct {
	Hash           string      `json:"hash"`
	Original       string      `json:"original"`
	Compressed     string      `json:"compressed,omitempty"` // Not stored if encrypted
	Command        string      `json:"command,omitempty"`
	ContentType    ContentType `json:"content_type"`
	CompressionAlg string      `json:"compression_alg"` // "zstd", "lz4", "none"
	Encrypted      bool        `json:"encrypted"`
	CreatedAt      time.Time   `json:"created_at"`
	AccessedAt     time.Time   `json:"accessed_at"`
	AccessCount    int         `json:"access_count"`
	SizeOriginal   int64       `json:"size_original"`
	SizeCompressed int64       `json:"size_compressed"`
	// These fields are only populated when retrieving, not stored in DB
	CompressedData []byte `json:"-"` // Raw compressed bytes (encrypted if Encrypted=true)
}

// IsExpired checks if entry is older than 30 days.
func (e *Entry) IsExpired() bool {
	return time.Since(e.CreatedAt) > 30*24*time.Hour
}

// CompressionRatio returns the compression ratio.
func (e *Entry) CompressionRatio() float64 {
	if e.SizeOriginal == 0 {
		return 1.0
	}
	return float64(e.SizeCompressed) / float64(e.SizeOriginal)
}

// SavedPercentage returns percentage of space saved.
func (e *Entry) SavedPercentage() float64 {
	if e.SizeOriginal == 0 {
		return 0
	}
	return (1.0 - e.CompressionRatio()) * 100
}

// Marker represents a hash marker for referencing compressed content.
type Marker struct {
	Hash     string // First 16 chars of SHA-256
	FullHash string // Full 64-char hash for verification
}

// String returns the marker in [rewind:hash16] format.
func (m Marker) String() string {
	return fmt.Sprintf("[rewind:%s]", m.Hash)
}

// ParseMarker parses a marker string in [rewind:hash16] format.
func ParseMarker(s string) (Marker, error) {
	if len(s) < 10 || s[0] != '[' || s[len(s)-1] != ']' {
		return Marker{}, fmt.Errorf("invalid marker format")
	}

	content := s[1 : len(s)-1]
	parts := splitMarker(content)
	if len(parts) != 2 || parts[0] != "rewind" {
		return Marker{}, fmt.Errorf("invalid marker format: expected [rewind:hash16]")
	}

	hash := parts[1]
	if len(hash) != 16 {
		return Marker{}, fmt.Errorf("invalid hash length: expected 16 chars, got %d", len(hash))
	}

	return Marker{Hash: hash}, nil
}

func splitMarker(s string) []string {
	var parts []string
	var current strings.Builder
	for _, r := range s {
		if r == ':' {
			parts = append(parts, current.String())
			current.Reset()
		} else {
			current.WriteRune(r)
		}
	}
	parts = append(parts, current.String())
	return parts
}

// ComputeHash computes the full SHA-256 hash.
func ComputeHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

// ComputeShortHash computes the 16-char short hash for markers.
func ComputeShortHash(content string) string {
	return ComputeHash(content)[:16]
}

// Store provides persistent storage for reversible compression.
type Store interface {
	// Save stores an entry and returns its hash.
	Save(entry *Entry) (string, error)
	// Retrieve gets an entry by hash (short or full).
	Retrieve(hash string) (*Entry, error)
	// Delete removes an entry.
	Delete(hash string) error
	// List returns all entries, optionally filtered.
	List(filter ListFilter) ([]*Entry, error)
	// Stats returns store statistics.
	Stats() (StoreStats, error)
	// Vacuum reclaims space.
	Vacuum() error
}

// ListFilter provides filtering for List operations.
type ListFilter struct {
	Command     string
	ContentType ContentType
	Since       time.Time
	Before      time.Time
	MinSize     int64
	MaxSize     int64
	Limit       int
}

// StoreStats provides statistics about the store.
type StoreStats struct {
	TotalEntries  int64
	TotalSizeOrig int64
	TotalSizeComp int64
	OldestEntry   time.Time
	NewestEntry   time.Time
	ByContentType map[ContentType]int64
	ByCommand     map[string]int64
}

// Config provides configuration for the reversible compression system.
type Config struct {
	// StorePath is the path to the SQLite database.
	StorePath string
	// EncryptionKey is the AES-256-GCM key (nil for no encryption).
	EncryptionKey []byte
	// DefaultAlgorithm is the default compression algorithm.
	DefaultAlgorithm string // "zstd", "lz4", "none"
	// MaxEntrySize is the maximum size for a single entry (default 100MB).
	MaxEntrySize int64
	// AutoVacuum enables automatic vacuum after deletes.
	AutoVacuum bool
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		StorePath:        filepath.Join(rewindDataPath(), "rewind.db"),
		DefaultAlgorithm: "zstd",
		MaxEntrySize:     100 * 1024 * 1024, // 100MB
		AutoVacuum:       true,
	}
}

func rewindDataPath() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "tokman")
	}

	if runtime.GOOS == "windows" {
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return filepath.Join(localAppData, "tokman")
		}
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "tokman", "data")
		}
	}

	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "tokman")
	}

	return filepath.Join(os.TempDir(), "tokman-data")
}

// Compressor provides compression functionality.
type Compressor interface {
	// Compress compresses data and returns the compressed bytes.
	Compress(data []byte) ([]byte, error)
	// Decompress decompresses data.
	Decompress(data []byte) ([]byte, error)
	// Name returns the compressor name.
	Name() string
}

// Encryptor provides encryption functionality.
type Encryptor interface {
	// Encrypt encrypts data with the given key.
	Encrypt(plaintext []byte) ([]byte, error)
	// Decrypt decrypts data with the given key.
	Decrypt(ciphertext []byte) ([]byte, error)
}

// AESEncryptor implements AES-256-GCM encryption.
type AESEncryptor struct {
	key []byte
}

// NewAESEncryptor creates a new AES encryptor.
func NewAESEncryptor(key []byte) (*AESEncryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("AES-256 requires 32-byte key, got %d", len(key))
	}
	return &AESEncryptor{key: key}, nil
}

// Encrypt implements Encryptor.
func (e *AESEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt implements Encryptor.
func (e *AESEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// ClassificationResult contains content classification info.
type ClassificationResult struct {
	Type       ContentType
	Confidence float64
	Language   string
}

// Classifier provides content classification.
type Classifier interface {
	// Classify determines the content type.
	Classify(content string) ClassificationResult
}

// SimpleClassifier implements basic content classification.
type SimpleClassifier struct{}

// Classify implements Classifier.
func (c *SimpleClassifier) Classify(content string) ClassificationResult {
	// Simple heuristic classification
	if looksLikeCode(content) {
		return ClassificationResult{Type: TypeLossless, Confidence: 0.9}
	}
	if looksLikeLogs(content) {
		return ClassificationResult{Type: TypeLossy, Confidence: 0.8}
	}
	return ClassificationResult{Type: TypeSemantic, Confidence: 0.7}
}

func looksLikeCode(content string) bool {
	codeIndicators := []string{"func ", "def ", "class ", "import ", "package ", "{", "}"}
	for _, ind := range codeIndicators {
		if contains(content, ind) {
			return true
		}
	}
	return false
}

func looksLikeLogs(content string) bool {
	logIndicators := []string{"[INFO]", "[ERROR]", "[WARN]", "timestamp", "level="}
	for _, ind := range logIndicators {
		if contains(content, ind) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// StreamingReader provides streaming decompression.
type StreamingReader interface {
	// Read reads decompressed data.
	Read(p []byte) (n int, err error)
	// Close closes the reader.
	Close() error
}

// CommandRecord stores command execution history.
type CommandRecord struct {
	Command    string        `json:"command"`
	Hash       string        `json:"hash"`
	Timestamp  time.Time     `json:"timestamp"`
	Duration   time.Duration `json:"duration"`
	Compressed bool          `json:"compressed"`
}
