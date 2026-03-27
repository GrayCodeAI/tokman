package filter

import (
	"strings"
	"testing"
)

// ── ANSIStripFilter ──────────────────────────────────────────────────────────

func TestANSIStripFilter_Name(t *testing.T) {
	f := NewANSIStripFilter()
	if f.Name() != "ansi_strip" {
		t.Errorf("Name() = %q, want %q", f.Name(), "ansi_strip")
	}
}

func TestANSIStripFilter_CSI(t *testing.T) {
	f := NewANSIStripFilter()
	// CSI color codes
	input := "\x1b[32mHello\x1b[0m World"
	output, _ := f.Apply(input, ModeMinimal)
	if strings.Contains(output, "\x1b[") {
		t.Error("CSI sequences should be stripped")
	}
	if !strings.Contains(output, "Hello") {
		t.Error("text content should be preserved")
	}
	if !strings.Contains(output, "World") {
		t.Error("text content after sequence should be preserved")
	}
}

func TestANSIStripFilter_OSC(t *testing.T) {
	f := NewANSIStripFilter()
	// OSC window title sequence
	input := "\x1b]0;My Terminal\x07plain text"
	output, _ := f.Apply(input, ModeMinimal)
	if strings.Contains(output, "\x1b]") {
		t.Error("OSC sequences should be stripped")
	}
	if !strings.Contains(output, "plain text") {
		t.Error("text after OSC sequence should be preserved")
	}
}

func TestANSIStripFilter_Charset(t *testing.T) {
	f := NewANSIStripFilter()
	// Charset select sequences (VT100 G0/G1)
	input := "\x1b(Bsome text\x1b)0"
	output, _ := f.Apply(input, ModeMinimal)
	if strings.Contains(output, "\x1b(") {
		t.Error("charset sequences should be stripped")
	}
	if strings.Contains(output, "\x1b)") {
		t.Error("charset sequences should be stripped")
	}
	if !strings.Contains(output, "some text") {
		t.Error("text should be preserved after stripping charset sequences")
	}
}

func TestANSIStripFilter_MultipleSequences(t *testing.T) {
	f := NewANSIStripFilter()
	input := "\x1b[1m\x1b[31mERROR\x1b[0m: something went wrong"
	output, saved := f.Apply(input, ModeAggressive)
	if strings.Contains(output, "\x1b[") {
		t.Error("all CSI sequences should be stripped")
	}
	if !strings.Contains(output, "ERROR") {
		t.Error("ERROR text should be preserved")
	}
	if saved <= 0 {
		t.Error("should report tokens saved when sequences are stripped")
	}
}

func TestANSIStripFilter_PlainTextUnchanged(t *testing.T) {
	f := NewANSIStripFilter()
	input := "This is plain text without any ANSI codes."
	output, saved := f.Apply(input, ModeMinimal)
	if output != input {
		t.Errorf("plain text should be unchanged, got %q", output)
	}
	if saved != 0 {
		t.Errorf("no sequences stripped means 0 saved, got %d", saved)
	}
}

func TestANSIStripFilter_EmptyInput(t *testing.T) {
	f := NewANSIStripFilter()
	output, saved := f.Apply("", ModeMinimal)
	if output != "" {
		t.Errorf("empty input should produce empty output, got %q", output)
	}
	if saved != 0 {
		t.Errorf("empty input should save 0, got %d", saved)
	}
}

func TestANSIStripFilter_RunsInModeNone(t *testing.T) {
	f := NewANSIStripFilter()
	// ANSIStripFilter strips in all modes, including ModeNone
	input := "\x1b[32mtext\x1b[0m"
	output, _ := f.Apply(input, ModeNone)
	if strings.Contains(output, "\x1b[") {
		t.Error("ANSI sequences should be stripped even in ModeNone")
	}
}

// ── WhitespaceNormalizer ─────────────────────────────────────────────────────

func TestWhitespaceNormalizer_Name(t *testing.T) {
	f := NewWhitespaceNormalizer()
	if f.Name() != "whitespace" {
		t.Errorf("Name() = %q, want %q", f.Name(), "whitespace")
	}
}

func TestWhitespaceNormalizer_ModeNone(t *testing.T) {
	f := NewWhitespaceNormalizer()
	input := "line1\n\n\n\n\nline2"
	output, saved := f.Apply(input, ModeNone)
	if output != input {
		t.Error("ModeNone should return input unchanged")
	}
	if saved != 0 {
		t.Errorf("ModeNone should save 0, got %d", saved)
	}
}

func TestWhitespaceNormalizer_MinimalCollapsesBlanks(t *testing.T) {
	f := NewWhitespaceNormalizer()
	// 4 blank lines -> ModeMinimal should collapse to 2
	input := "line1\n\n\n\n\nline2"
	output, _ := f.Apply(input, ModeMinimal)
	lines := strings.Split(output, "\n")
	// Count consecutive blank lines
	maxConsecutive := 0
	consecutive := 0
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			consecutive++
			if consecutive > maxConsecutive {
				maxConsecutive = consecutive
			}
		} else {
			consecutive = 0
		}
	}
	if maxConsecutive > 2 {
		t.Errorf("ModeMinimal should allow at most 2 consecutive blank lines, got %d", maxConsecutive)
	}
	if !strings.Contains(output, "line1") || !strings.Contains(output, "line2") {
		t.Error("content lines should be preserved")
	}
}

func TestWhitespaceNormalizer_AggressiveCollapsesBlanks(t *testing.T) {
	f := NewWhitespaceNormalizer()
	// 3 blank lines -> ModeAggressive should collapse to 1
	input := "line1\n\n\n\nline2"
	output, _ := f.Apply(input, ModeAggressive)
	lines := strings.Split(output, "\n")
	maxConsecutive := 0
	consecutive := 0
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			consecutive++
			if consecutive > maxConsecutive {
				maxConsecutive = consecutive
			}
		} else {
			consecutive = 0
		}
	}
	if maxConsecutive > 1 {
		t.Errorf("ModeAggressive should allow at most 1 consecutive blank line, got %d", maxConsecutive)
	}
}

func TestWhitespaceNormalizer_TrailingWhitespaceStripped(t *testing.T) {
	f := NewWhitespaceNormalizer()
	input := "line with trailing spaces   \nline with tab\t\nclean line"
	output, _ := f.Apply(input, ModeMinimal)
	for _, line := range strings.Split(output, "\n") {
		if strings.HasSuffix(line, " ") || strings.HasSuffix(line, "\t") {
			t.Errorf("trailing whitespace not stripped from line: %q", line)
		}
	}
}

func TestWhitespaceNormalizer_TabExpansion(t *testing.T) {
	f := NewWhitespaceNormalizer()
	input := "\tindented"
	output, _ := f.Apply(input, ModeMinimal)
	if strings.Contains(output, "\t") {
		t.Error("tabs should be expanded to spaces")
	}
	if !strings.Contains(output, "    indented") {
		t.Errorf("tab should expand to 4 spaces, got: %q", output)
	}
}

func TestWhitespaceNormalizer_AggressiveSavesMore(t *testing.T) {
	f := NewWhitespaceNormalizer()
	// Aggressive mode should collapse more blank lines, thus saving more tokens
	input := strings.Repeat("line\n\n\n", 20)
	_, savedMinimal := f.Apply(input, ModeMinimal)
	_, savedAggressive := f.Apply(input, ModeAggressive)
	if savedAggressive < savedMinimal {
		t.Error("aggressive mode should save at least as many tokens as minimal mode")
	}
}

// ── CommentStripFilter ───────────────────────────────────────────────────────

func TestCommentStripFilter_Name(t *testing.T) {
	f := NewCommentStripFilter()
	if f.Name() != "comment_strip" {
		t.Errorf("Name() = %q, want %q", f.Name(), "comment_strip")
	}
}

func TestCommentStripFilter_SlashSlashComments(t *testing.T) {
	f := NewCommentStripFilter()
	input := "// this is a comment\nfunc hello() {}\n// another comment"
	output, saved := f.Apply(input, ModeMinimal)
	if strings.Contains(output, "this is a comment") {
		t.Error("// comments should be stripped")
	}
	if strings.Contains(output, "another comment") {
		t.Error("// comments should be stripped")
	}
	if !strings.Contains(output, "func hello") {
		t.Error("code should be preserved")
	}
	if saved <= 0 {
		t.Error("stripping comments should save tokens")
	}
}

func TestCommentStripFilter_HashComments(t *testing.T) {
	f := NewCommentStripFilter()
	input := "# this is a python comment\nprint('hello')\n# another comment"
	output, _ := f.Apply(input, ModeMinimal)
	if strings.Contains(output, "this is a python comment") {
		t.Error("# comments should be stripped")
	}
	if !strings.Contains(output, "print") {
		t.Error("code should be preserved")
	}
}

func TestCommentStripFilter_DashDashComments(t *testing.T) {
	f := NewCommentStripFilter()
	input := "-- SQL comment\nSELECT * FROM table;\n-- another comment"
	output, _ := f.Apply(input, ModeMinimal)
	if strings.Contains(output, "SQL comment") {
		t.Error("-- comments should be stripped")
	}
	if !strings.Contains(output, "SELECT") {
		t.Error("SQL code should be preserved")
	}
}

func TestCommentStripFilter_BlockComments(t *testing.T) {
	f := NewCommentStripFilter()
	input := "/* block comment */\nint x = 5;\n/* another\nmultiline\nblock */"
	output, saved := f.Apply(input, ModeMinimal)
	if strings.Contains(output, "block comment") {
		t.Error("/* */ block comments should be stripped")
	}
	if !strings.Contains(output, "int x = 5") {
		t.Error("code should be preserved after block comment stripping")
	}
	if saved <= 0 {
		t.Error("stripping block comments should save tokens")
	}
}

func TestCommentStripFilter_StringLiteralsPreserved(t *testing.T) {
	f := NewCommentStripFilter()
	// The // inside a string literal must NOT be stripped
	input := `url := "https://example.com/path"` + "\n" + `// real comment`
	output, _ := f.Apply(input, ModeMinimal)
	if !strings.Contains(output, "https://example.com/path") {
		t.Error("string literal containing // should be preserved")
	}
	if strings.Contains(output, "real comment") {
		t.Error("actual comment should be stripped")
	}
}

func TestCommentStripFilter_ModeNoneReturnsUnchanged(t *testing.T) {
	f := NewCommentStripFilter()
	input := "// comment\nfunc foo() {}"
	output, saved := f.Apply(input, ModeNone)
	if output != input {
		t.Error("ModeNone should return input unchanged")
	}
	if saved != 0 {
		t.Errorf("ModeNone should save 0, got %d", saved)
	}
}

func TestCommentStripFilter_HTMLComments(t *testing.T) {
	f := NewCommentStripFilter()
	input := "<!-- this is an HTML comment -->\n<div>content</div>"
	output, _ := f.Apply(input, ModeMinimal)
	if strings.Contains(output, "this is an HTML comment") {
		t.Error("HTML comments should be stripped")
	}
	if !strings.Contains(output, "content") {
		t.Error("HTML content should be preserved")
	}
}

func TestCommentStripFilter_PreserveDocComments(t *testing.T) {
	f := NewCommentStripFilterPreserveDoc()
	input := "/// doc comment\n// regular comment\nfunc foo() {}"
	output, _ := f.Apply(input, ModeMinimal)
	if !strings.Contains(output, "doc comment") {
		t.Error("doc comments (///) should be preserved when PreserveDocComments is true")
	}
}

// ── ChunkBoundaryDetector ────────────────────────────────────────────────────

func TestChunkBoundaryDetector_SplitAtBlankLines(t *testing.T) {
	d := NewChunkBoundaryDetector(50) // small target to force splitting
	// Build a block of lines to ensure splitting occurs
	block1 := strings.Repeat("line of content here\n", 15)
	block2 := strings.Repeat("more content here\n", 15)
	input := block1 + "\n" + block2
	chunks := d.SplitAtBoundaries(input)
	if len(chunks) < 2 {
		t.Errorf("expected at least 2 chunks from splitting at blank line, got %d", len(chunks))
	}
}

func TestChunkBoundaryDetector_FallsBackToClosingBrace(t *testing.T) {
	d := NewChunkBoundaryDetector(50)
	// Content with closing brace at column 0 but no blank lines
	block := strings.Repeat("    body line here abc\n", 15)
	input := "func foo() {\n" + block + "}\n" + strings.Repeat("    more lines\n", 15)
	chunks := d.SplitAtBoundaries(input)
	// Should split at the closing brace
	if len(chunks) < 1 {
		t.Error("expected chunks to be produced")
	}
	// At least one chunk should end with or contain the closing brace
	found := false
	for _, ch := range chunks {
		if strings.Contains(ch, "}") {
			found = true
			break
		}
	}
	if !found {
		t.Error("closing brace should appear in one of the chunks")
	}
}

func TestChunkBoundaryDetector_SmallInputNotSplit(t *testing.T) {
	d := NewChunkBoundaryDetector(2000)
	input := "small input\nfits in one chunk\n"
	chunks := d.SplitAtBoundaries(input)
	if len(chunks) != 1 {
		t.Errorf("small input should produce 1 chunk, got %d", len(chunks))
	}
}

func TestChunkBoundaryDetector_EmptyInput(t *testing.T) {
	d := NewChunkBoundaryDetector(2000)
	chunks := d.SplitAtBoundaries("")
	if len(chunks) != 0 {
		t.Errorf("empty input should produce 0 chunks, got %d", len(chunks))
	}
}

func TestChunkBoundaryDetector_DefaultTargetTokens(t *testing.T) {
	d := NewChunkBoundaryDetector(0) // 0 => uses default 2000
	if d.TargetChunkTokens != 2000 {
		t.Errorf("default TargetChunkTokens = %d, want 2000", d.TargetChunkTokens)
	}
}

func TestChunkBoundaryDetector_AllChunksNonEmpty(t *testing.T) {
	d := NewChunkBoundaryDetector(100)
	input := strings.Repeat("this is a line of content that is fairly long\n", 30)
	chunks := d.SplitAtBoundaries(input)
	for i, ch := range chunks {
		if strings.TrimSpace(ch) == "" {
			t.Errorf("chunk %d is empty or whitespace-only", i)
		}
	}
}

// ── SensitivityClassifier ────────────────────────────────────────────────────

func TestSensitivityClassifier_DetectsAWSKey(t *testing.T) {
	c := NewSensitivityClassifier()
	input := "export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE"
	matches := c.Classify(input)
	if len(matches) == 0 {
		t.Error("should detect AWS access key")
	}
	found := false
	for _, m := range matches {
		if m.Category == SensitivityCredentials {
			found = true
			break
		}
	}
	if !found {
		t.Error("AWS key should be classified as credentials")
	}
}

func TestSensitivityClassifier_DetectsEmail(t *testing.T) {
	c := NewSensitivityClassifier()
	input := "Contact us at user@example.com for support."
	matches := c.Classify(input)
	if len(matches) == 0 {
		t.Error("should detect email address")
	}
	found := false
	for _, m := range matches {
		if m.Category == SensitivityPII {
			found = true
			break
		}
	}
	if !found {
		t.Error("email should be classified as PII")
	}
}

func TestSensitivityClassifier_DetectsSSN(t *testing.T) {
	c := NewSensitivityClassifier()
	input := "Patient SSN: 123-45-6789"
	matches := c.Classify(input)
	if len(matches) == 0 {
		t.Error("should detect US SSN")
	}
	found := false
	for _, m := range matches {
		if m.Category == SensitivityPII {
			found = true
			break
		}
	}
	if !found {
		t.Error("SSN should be classified as PII")
	}
}

func TestSensitivityClassifier_CleanInputReturnsEmpty(t *testing.T) {
	c := NewSensitivityClassifier()
	input := "This is completely clean text with no sensitive data whatsoever."
	matches := c.Classify(input)
	if len(matches) != 0 {
		t.Errorf("clean input should return 0 matches, got %d: %v", len(matches), matches)
	}
}

func TestSensitivityClassifier_HasSensitiveContent(t *testing.T) {
	c := NewSensitivityClassifier()
	if !c.HasSensitiveContent("email: test@example.com") {
		t.Error("should detect sensitive content")
	}
	if c.HasSensitiveContent("no sensitive data here") {
		t.Error("clean text should not be detected as sensitive")
	}
}

func TestSensitivityClassifier_SnippetRedacted(t *testing.T) {
	c := NewSensitivityClassifier()
	input := "AKIAIOSFODNN7EXAMPLE is a key"
	matches := c.Classify(input)
	if len(matches) == 0 {
		t.Fatal("should find a match")
	}
	if strings.Contains(matches[0].Snippet, "AKIAIOSFODNN7EXAMPLE") {
		t.Error("snippet should redact the sensitive value")
	}
	if !strings.Contains(matches[0].Snippet, "[REDACTED]") {
		t.Error("snippet should contain [REDACTED] marker")
	}
}

func TestSensitivityClassifier_Categories(t *testing.T) {
	c := NewSensitivityClassifier()
	input := "email: user@test.com\nAKIAIOSFODNN7EXAMPLE"
	cats := c.Categories(input)
	if len(cats) == 0 {
		t.Error("should find at least one category")
	}
}

func TestSensitivityClassifier_LineNumbers(t *testing.T) {
	c := NewSensitivityClassifier()
	input := "clean line\nuser@example.com"
	matches := c.Classify(input)
	if len(matches) == 0 {
		t.Fatal("should find email match")
	}
	if matches[0].LineNum != 2 {
		t.Errorf("email is on line 2, got LineNum=%d", matches[0].LineNum)
	}
}

// ── AdaptiveContextWindowCompressor ─────────────────────────────────────────

func TestAdaptiveContextWindowCompressor_Name(t *testing.T) {
	a := NewAdaptiveContextWindowCompressor("gpt-4o")
	if a.Name() != "adaptive_context_window" {
		t.Errorf("Name() = %q, want %q", a.Name(), "adaptive_context_window")
	}
}

func TestAdaptiveContextWindowCompressor_SmallWindowTier(t *testing.T) {
	// gpt-3.5-turbo has 16K window → should be TierSmall
	a := NewAdaptiveContextWindowCompressor("gpt-3.5-turbo")
	profile := a.Profile()
	if profile.Tier != TierSmall {
		t.Errorf("gpt-3.5-turbo should be TierSmall, got %q", profile.Tier)
	}
}

func TestAdaptiveContextWindowCompressor_MediumWindowTier(t *testing.T) {
	// gpt-4o has 128K window → should be TierMedium
	a := NewAdaptiveContextWindowCompressor("gpt-4o")
	profile := a.Profile()
	if profile.Tier != TierMedium {
		t.Errorf("gpt-4o should be TierMedium, got %q", profile.Tier)
	}
}

func TestAdaptiveContextWindowCompressor_LargeWindowTier(t *testing.T) {
	// claude-sonnet-4.6 has 200K window → should be TierLarge
	a := NewAdaptiveContextWindowCompressor("claude-sonnet-4.6")
	profile := a.Profile()
	if profile.Tier != TierLarge {
		t.Errorf("claude-sonnet-4.6 should be TierLarge, got %q", profile.Tier)
	}
}

func TestAdaptiveContextWindowCompressor_XLargeWindowTier(t *testing.T) {
	// gemini-1.5-pro has 1M window → should be TierXLarge
	a := NewAdaptiveContextWindowCompressor("gemini-1.5-pro")
	profile := a.Profile()
	if profile.Tier != TierXLarge {
		t.Errorf("gemini-1.5-pro should be TierXLarge, got %q", profile.Tier)
	}
}

func TestAdaptiveContextWindowCompressor_ExplicitWindowSize(t *testing.T) {
	// Explicit small window override
	a := NewAdaptiveContextWindowCompressorWithSize(8000)
	profile := a.Profile()
	if profile.Tier != TierSmall {
		t.Errorf("8000 token window should be TierSmall, got %q", profile.Tier)
	}
}

func TestAdaptiveContextWindowCompressor_ProfileForWindow(t *testing.T) {
	tests := []struct {
		tokens int
		tier   ContextWindowTier
	}{
		{4000, TierSmall},
		{32000, TierMedium},
		{200000, TierLarge},
		{1000000, TierXLarge},
	}
	for _, tt := range tests {
		profile := ProfileForWindow(tt.tokens)
		if profile.Tier != tt.tier {
			t.Errorf("ProfileForWindow(%d).Tier = %q, want %q", tt.tokens, profile.Tier, tt.tier)
		}
	}
}

func TestAdaptiveContextWindowCompressor_UnknownModelDefaultsMedium(t *testing.T) {
	a := NewAdaptiveContextWindowCompressor("unknown-model-xyz")
	profile := a.Profile()
	// Unknown model falls back to 128K default → TierMedium
	if profile.Tier != TierMedium {
		t.Errorf("unknown model should default to TierMedium, got %q", profile.Tier)
	}
}

func TestAdaptiveContextWindowCompressor_XLargePassthrough(t *testing.T) {
	a := NewAdaptiveContextWindowCompressor("gemini-1.5-pro")
	input := "some content to compress"
	output, saved := a.Apply(input, ModeMinimal)
	// XLarge window → ModeNone → no compression
	if output != input {
		t.Error("XLarge tier should pass through input unchanged")
	}
	if saved != 0 {
		t.Errorf("XLarge tier should save 0 tokens, got %d", saved)
	}
}
