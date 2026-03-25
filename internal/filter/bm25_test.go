package filter

import (
	"testing"
)

func TestBM25ScorerNew(t *testing.T) {
	s := newBM25Scorer()
	if s == nil {
		t.Fatal("newBM25Scorer returned nil")
	}
	if s.k1 != 1.2 {
		t.Errorf("k1 = %f, want 1.2", s.k1)
	}
	if s.b != 0.75 {
		t.Errorf("b = %f, want 0.75", s.b)
	}
}

func TestBM25Scorer_Fit(t *testing.T) {
	s := newBM25Scorer()
	docs := []string{
		"the quick brown fox",
		"the lazy dog sleeps",
		"brown fox jumps high",
	}
	s.Fit(docs)
	if s.docCount != 3 {
		t.Errorf("docCount = %d, want 3", s.docCount)
	}
	if s.avgDocLength <= 0 {
		t.Errorf("avgDocLength = %f, want > 0", s.avgDocLength)
	}
}

func TestBM25Scorer_Score(t *testing.T) {
	s := newBM25Scorer()
	docs := []string{
		"error: connection failed at 192.168.1.1",
		"info: server started on port 8080",
		"error: timeout waiting for response",
		"debug: processing request",
	}
	s.Fit(docs)

	errorScore := s.Score(docs[0], "error connection")
	infoScore := s.Score(docs[1], "error connection")

	if errorScore <= infoScore {
		t.Errorf("error doc score %f should be > info doc score %f for 'error connection' query", errorScore, infoScore)
	}
}

func TestBM25Scorer_ScoreLines(t *testing.T) {
	s := newBM25Scorer()
	lines := []string{
		"ERROR: database connection failed",
		"INFO: starting application",
		"ERROR: query timeout",
		"DEBUG: cache hit",
	}
	scores := s.ScoreLines(lines, "ERROR database")
	if len(scores) != 4 {
		t.Fatalf("expected 4 scores, got %d", len(scores))
	}
	// First line should score highest (contains both ERROR and database)
	if scores[0].Index != 0 {
		t.Errorf("highest scoring line index = %d, want 0", scores[0].Index)
	}
}

func TestBM25Scorer_UnseenTerm(t *testing.T) {
	s := newBM25Scorer()
	docs := []string{"hello world", "foo bar"}
	s.Fit(docs)
	score := s.Score("hello world", "nonexistent")
	if score < 0 {
		t.Errorf("score for unseen term should be >= 0, got %f", score)
	}
}

func TestQuestionAwareRecovery(t *testing.T) {
	qr := newQuestionAwareRecovery()
	original := "line1\nERROR: failed\nline3\nline4"
	compressed := "line1\nline3\nline4"
	recovered := qr.Recover(original, compressed, "ERROR failed")
	// Recovery returns a string (may or may not add lines depending on scoring)
	if recovered == "" {
		t.Error("recovered output should not be empty")
	}
}

func TestQuestionAwareRecovery_NoRecovery(t *testing.T) {
	qr := newQuestionAwareRecovery()
	original := "line1\nline2\nline3"
	compressed := "line1\nline2\nline3"
	recovered := qr.Recover(original, compressed, "query")
	// All lines already present, nothing to recover
	if recovered != compressed {
		t.Errorf("should not add lines when all present")
	}
}
