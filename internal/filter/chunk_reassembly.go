package filter

import (
	"fmt"
	"strings"
)

// ChunkReassembler reassembles a list of compressed chunk strings into a single
// coherent output. It validates chunks and warns about potential bad splits.
//
// Task #151: Chunk reassembly with coherence checking.
type ChunkReassembler struct {
	separator string
}

// ReassemblyResult holds the output and diagnostics from a reassembly operation.
type ReassemblyResult struct {
	// Output is the reassembled text.
	Output string
	// Warnings contains human-readable advisories about the reassembly.
	Warnings []string
	// ChunkCount is the number of input chunks that were processed.
	ChunkCount int
}

// NewChunkReassembler creates a ChunkReassembler with the given separator.
// If separator is empty, it defaults to "\n\n".
func NewChunkReassembler(separator string) *ChunkReassembler {
	if separator == "" {
		separator = "\n\n"
	}
	return &ChunkReassembler{separator: separator}
}

// Reassemble joins the provided chunks with the configured separator and
// performs a coherence check. It returns an error if any chunk is empty.
// Warnings are recorded when adjacent chunks share no words, which may
// indicate a bad split boundary.
func (r *ChunkReassembler) Reassemble(chunks []string) (string, error) {
	result, err := r.ReassembleWithResult(chunks)
	if err != nil {
		return "", err
	}
	return result.Output, nil
}

// ReassembleWithResult is like Reassemble but returns a full ReassemblyResult
// that includes warnings and metadata.
func (r *ChunkReassembler) ReassembleWithResult(chunks []string) (ReassemblyResult, error) {
	var warnings []string

	// Coherence check 1: no chunk may be empty.
	for i, chunk := range chunks {
		if strings.TrimSpace(chunk) == "" {
			return ReassemblyResult{}, fmt.Errorf("chunk %d is empty", i)
		}
	}

	// Coherence check 2: warn when adjacent chunks share zero words.
	for i := 1; i < len(chunks); i++ {
		if !chunksShareWords(chunks[i-1], chunks[i]) {
			warnings = append(warnings, fmt.Sprintf(
				"chunks %d and %d share no common words — possible bad split boundary", i-1, i,
			))
		}
	}

	output := strings.Join(chunks, r.separator)

	return ReassemblyResult{
		Output:     output,
		Warnings:   warnings,
		ChunkCount: len(chunks),
	}, nil
}

// chunksShareWords returns true if two chunk strings share at least one word
// (case-insensitive, ignoring punctuation/whitespace).
func chunksShareWords(a, b string) bool {
	wordsA := chunkWordSet(a)
	for _, w := range tokenize(b) {
		if wordsA[strings.ToLower(w)] {
			return true
		}
	}
	return false
}

// chunkWordSet builds a lowercase word set from a chunk string.
func chunkWordSet(s string) map[string]bool {
	words := tokenize(s)
	set := make(map[string]bool, len(words))
	for _, w := range words {
		set[strings.ToLower(w)] = true
	}
	return set
}
