package filter

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

// E2EEncryptedResult wraps compressed output with AES-256-GCM encryption.
// The encryption key is derived from a passphrase using SHA-256.
// Task #193: End-to-end encryption for compression API.
type E2EEncryptedResult struct {
	// Ciphertext is the base64-encoded encrypted content.
	Ciphertext string
	// Nonce is the base64-encoded GCM nonce (12 bytes).
	Nonce string
	// OriginalTokens (metadata, unencrypted for budgeting).
	OriginalTokens int
	// CompressedTokens (metadata, unencrypted for budgeting).
	CompressedTokens int
}

// Format serializes the result for transmission.
func (r *E2EEncryptedResult) Format() string {
	return fmt.Sprintf("[tokman-enc:aes256gcm nonce=%s orig=%d comp=%d]\n%s",
		r.Nonce, r.OriginalTokens, r.CompressedTokens, r.Ciphertext)
}

// E2EEncryptor encrypts and decrypts compression results.
type E2EEncryptor struct {
	key [32]byte
}

// NewE2EEncryptor creates an encryptor keyed from passphrase.
// The passphrase is hashed with SHA-256 to produce a 256-bit key.
func NewE2EEncryptor(passphrase string) *E2EEncryptor {
	key := sha256.Sum256([]byte(passphrase))
	return &E2EEncryptor{key: key}
}

// Encrypt encrypts plaintext and returns the encrypted result.
func (e *E2EEncryptor) Encrypt(plaintext string, origTokens, compTokens int) (*E2EEncryptedResult, error) {
	block, err := aes.NewCipher(e.key[:])
	if err != nil {
		return nil, fmt.Errorf("e2e encrypt: cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("e2e encrypt: gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("e2e encrypt: nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return &E2EEncryptedResult{
		Ciphertext:       base64.StdEncoding.EncodeToString(ciphertext),
		Nonce:            base64.StdEncoding.EncodeToString(nonce),
		OriginalTokens:   origTokens,
		CompressedTokens: compTokens,
	}, nil
}

// Decrypt decrypts an encrypted result back to plaintext.
func (e *E2EEncryptor) Decrypt(r *E2EEncryptedResult) (string, error) {
	nonce, err := base64.StdEncoding.DecodeString(r.Nonce)
	if err != nil {
		return "", fmt.Errorf("e2e decrypt: nonce decode: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(r.Ciphertext)
	if err != nil {
		return "", fmt.Errorf("e2e decrypt: ciphertext decode: %w", err)
	}

	block, err := aes.NewCipher(e.key[:])
	if err != nil {
		return "", fmt.Errorf("e2e decrypt: cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("e2e decrypt: gcm: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("e2e decrypt: open: %w (wrong key or tampered data)", err)
	}
	return string(plaintext), nil
}

// DecryptFormatted parses and decrypts output produced by E2EEncryptedResult.Format().
func (e *E2EEncryptor) DecryptFormatted(formatted string) (string, error) {
	lines := strings.SplitN(formatted, "\n", 2)
	if len(lines) < 2 {
		return "", errors.New("e2e decrypt: invalid format")
	}
	header := lines[0]
	ciphertext := lines[1]

	var nonce string
	var orig, comp int
	_, err := fmt.Sscanf(header, "[tokman-enc:aes256gcm nonce=%s orig=%d comp=%d]",
		&nonce, &orig, &comp)
	// Sscanf may not parse the last `]` — strip it
	nonce = strings.TrimSuffix(nonce, "]")
	if err != nil {
		return "", fmt.Errorf("e2e decrypt: parse header: %w", err)
	}

	return e.Decrypt(&E2EEncryptedResult{
		Nonce:            nonce,
		Ciphertext:       ciphertext,
		OriginalTokens:   orig,
		CompressedTokens: comp,
	})
}

// EncryptedPipeline wraps a PipelineCoordinator with E2E encryption.
// Process() compresses and then encrypts the result.
type EncryptedPipeline struct {
	coordinator *PipelineCoordinator
	encryptor   *E2EEncryptor
}

// NewEncryptedPipeline creates an encrypted pipeline with the given passphrase.
func NewEncryptedPipeline(cfg PipelineConfig, passphrase string) *EncryptedPipeline {
	return &EncryptedPipeline{
		coordinator: NewPipelineCoordinator(cfg),
		encryptor:   NewE2EEncryptor(passphrase),
	}
}

// Process compresses input and returns the formatted encrypted output.
func (p *EncryptedPipeline) Process(input string) (string, *PipelineStats, error) {
	compressed, stats := p.coordinator.Process(input)
	encrypted, err := p.encryptor.Encrypt(compressed, stats.OriginalTokens, stats.FinalTokens)
	if err != nil {
		return "", stats, err
	}
	return encrypted.Format(), stats, nil
}
