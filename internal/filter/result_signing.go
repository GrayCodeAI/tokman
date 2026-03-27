package filter

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// SignedResult wraps a compression result with an integrity signature.
// The signature allows consumers to verify that the result was not tampered with
// and that it originated from this pipeline instance.
// Task #174: Compression result signing.
type SignedResult struct {
	// Output is the compressed content.
	Output string
	// OriginalTokens is the token count before compression.
	OriginalTokens int
	// CompressedTokens is the token count after compression.
	CompressedTokens int
	// Timestamp is when this result was produced.
	Timestamp time.Time
	// Signature is an HMAC-SHA256 signature over canonical fields.
	Signature string
	// PipelineID identifies the pipeline that produced this result.
	PipelineID string
}

// signingKey is the default HMAC key used when no key is provided.
// In production, override with a secret from config.
var defaultSigningKey = []byte("tokman-default-signing-key-v1")

// ResultSigner signs and verifies compression results.
type ResultSigner struct {
	key        []byte
	pipelineID string
}

// NewResultSigner creates a signer with the given HMAC key and pipeline ID.
// key may be nil (uses a built-in default; NOT suitable for security-sensitive use).
func NewResultSigner(key []byte, pipelineID string) *ResultSigner {
	if len(key) == 0 {
		key = defaultSigningKey
	}
	if pipelineID == "" {
		pipelineID = "tokman-pipeline"
	}
	return &ResultSigner{key: key, pipelineID: pipelineID}
}

// Sign produces a signed result for the given output and stats.
func (s *ResultSigner) Sign(output string, originalTokens, compressedTokens int) *SignedResult {
	ts := time.Now().UTC()
	r := &SignedResult{
		Output:           output,
		OriginalTokens:   originalTokens,
		CompressedTokens: compressedTokens,
		Timestamp:        ts,
		PipelineID:       s.pipelineID,
	}
	r.Signature = s.computeSignature(r)
	return r
}

// Verify checks that the result's signature matches its contents.
func (s *ResultSigner) Verify(r *SignedResult) bool {
	expected := s.computeSignature(r)
	return hmac.Equal([]byte(r.Signature), []byte(expected))
}

// computeSignature computes the HMAC-SHA256 of the canonical representation.
func (s *ResultSigner) computeSignature(r *SignedResult) string {
	// Canonical string: tab-separated fields, deterministic order
	canonical := strings.Join([]string{
		r.PipelineID,
		r.Timestamp.Format(time.RFC3339Nano),
		fmt.Sprintf("%d", r.OriginalTokens),
		fmt.Sprintf("%d", r.CompressedTokens),
		r.Output,
	}, "\t")

	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil))
}

// FormatHeader returns a compact header line for embedding in output.
// Format: `[tokman-sig: pipeline=X ts=T orig=N comp=N sig=XXXX]`
func (r *SignedResult) FormatHeader() string {
	return fmt.Sprintf("[tokman-sig: pipeline=%s ts=%s orig=%d comp=%d sig=%s]",
		r.PipelineID,
		r.Timestamp.Format(time.RFC3339),
		r.OriginalTokens,
		r.CompressedTokens,
		r.Signature[:16], // truncate for brevity
	)
}
