package filter

import (
	"crypto/sha256"
	"fmt"
	"unicode/utf8"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// BinaryPassthrough detects binary output and passes it through unchanged.
// Compression on binary data (images, compiled binaries, PDFs) is harmful
// and can corrupt the output. This filter detects binary content and skips
// all compression layers.
type BinaryPassthrough struct {
	config BinaryConfig
}

// BinaryConfig holds configuration for binary detection
type BinaryConfig struct {
	Enabled        bool
	MaxBinaryRatio float64 // Max ratio of non-UTF8 bytes before marking as binary
	MinCheckBytes  int     // Minimum bytes to check
}

// DefaultBinaryConfig returns default configuration
func DefaultBinaryConfig() BinaryConfig {
	return BinaryConfig{
		Enabled:        true,
		MaxBinaryRatio: 0.1, // >10% non-UTF8 = binary
		MinCheckBytes:  512,
	}
}

// NewBinaryPassthrough creates a new binary detector
func NewBinaryPassthrough() *BinaryPassthrough {
	return &BinaryPassthrough{
		config: DefaultBinaryConfig(),
	}
}

// Name returns the filter name
func (b *BinaryPassthrough) Name() string {
	return "binary_passthrough"
}

// IsBinary checks if content is binary (without modifying)
func (b *BinaryPassthrough) IsBinary(input string) bool {
	if !b.config.Enabled {
		return false
	}

	if len(input) == 0 {
		return false
	}

	// Quick check: null bytes are a strong binary indicator
	for i := 0; i < len(input) && i < b.config.MinCheckBytes; i++ {
		if input[i] == 0 {
			return true
		}
	}

	// Check UTF-8 validity ratio
	checkLen := len(input)
	if checkLen > b.config.MinCheckBytes*10 {
		checkLen = b.config.MinCheckBytes * 10
	}

	invalidCount := 0
	validCount := 0
	for i := 0; i < checkLen; {
		r, size := utf8.DecodeRuneInString(input[i:])
		if r == utf8.RuneError && size == 1 {
			invalidCount++
		} else {
			validCount++
		}
		i += size
	}

	total := invalidCount + validCount
	if total == 0 {
		return false
	}

	ratio := float64(invalidCount) / float64(total)
	return ratio > b.config.MaxBinaryRatio
}

// Apply emits a compact metadata summary when binary content is detected,
// saving significant tokens vs passing the raw bytes through.
// For text content, returns the input unchanged.
func (b *BinaryPassthrough) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone || !b.IsBinary(input) {
		return input, 0
	}

	original := core.EstimateTokens(input)

	// Detect file type from magic bytes
	fileType := detectMagicType(input)

	// Compute short hash for identity
	h := sha256.Sum256([]byte(input))
	hashStr := fmt.Sprintf("%x", h[:4]) // 8 hex chars

	summary := fmt.Sprintf("[binary-content: type=%s size=%d sha256-prefix=%s — raw bytes omitted]",
		fileType, len(input), hashStr)

	saved := original - core.EstimateTokens(summary)
	if saved <= 0 {
		return input, 0
	}
	return summary, saved
}

// detectMagicType identifies common file types by their magic bytes.
func detectMagicType(content string) string {
	if len(content) < 4 {
		return "unknown"
	}
	magic := []byte(content[:min10(len(content), 16)])

	switch {
	case len(magic) >= 4 && magic[0] == 0xFF && magic[1] == 0xD8 && magic[2] == 0xFF:
		return "jpeg"
	case len(magic) >= 8 && string(magic[:8]) == "\x89PNG\r\n\x1a\n":
		return "png"
	case len(magic) >= 4 && string(magic[:4]) == "GIF8":
		return "gif"
	case len(magic) >= 4 && string(magic[:4]) == "RIFF":
		return "riff" // WAV or AVI
	case len(magic) >= 4 && string(magic[:4]) == "%PDF":
		return "pdf"
	case len(magic) >= 4 && magic[0] == 0x7F && magic[1] == 'E' && magic[2] == 'L' && magic[3] == 'F':
		return "elf"
	case len(magic) >= 4 && magic[0] == 0x50 && magic[1] == 0x4B && magic[2] == 0x03:
		return "zip"
	case len(magic) >= 6 && string(magic[:6]) == "\x1f\x8b\x08":
		return "gzip"
	case len(magic) >= 4 && string(magic[:4]) == "MZ\x90\x00":
		return "pe-executable"
	default:
		return "binary"
	}
}
