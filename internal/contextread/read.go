package contextread

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/GrayCodeAI/tokman/internal/cache"
	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/graph"
)

var trackedCommandPatterns = []string{
	"tokman read *",
	"tokman ctx read *",
	"tokman ctx delta *",
	"tokman mcp read *",
}

var trackedCommandPatternsByKind = map[string][]string{
	"read":  {"tokman read *", "tokman ctx read *"},
	"delta": {"tokman ctx delta *"},
	"mcp":   {"tokman mcp read *"},
}

// TrackedCommandPatterns returns the command patterns used to summarize smart
// context-read activity in tracking views.
func TrackedCommandPatterns() []string {
	return append([]string(nil), trackedCommandPatterns...)
}

// TrackedCommandPatternsForKind returns command patterns for one smart-read kind.
func TrackedCommandPatternsForKind(kind string) []string {
	patterns, ok := trackedCommandPatternsByKind[strings.ToLower(kind)]
	if !ok {
		return nil
	}
	return append([]string(nil), patterns...)
}

// TrackedCommandKinds returns the supported smart-read categories.
func TrackedCommandKinds() []string {
	return []string{"read", "delta", "mcp"}
}

// Options control smart read rendering for files and MCP clients.
type Options struct {
	Level        string
	Mode         string
	MaxLines     int
	MaxTokens    int
	LineNumbers  bool
	StartLine    int
	EndLine      int
	SaveSnapshot bool
	RelatedFiles int
}

type buildResult struct {
	Output         string
	OriginalTokens int
	FinalTokens    int
}

var renderCache = cache.NewLRUCache(512, 10*time.Minute)

// Build renders content according to the requested read behavior.
func Build(filePath, content string, opts Options) (string, int, int, error) {
	if key, ok := cacheKey(filePath, content, opts); ok {
		if cached, ok := renderCache.Get(key).(buildResult); ok {
			return cached.Output, cached.OriginalTokens, cached.FinalTokens, nil
		}
	}

	filtered, err := ApplyMode(filePath, content, opts)
	if err != nil {
		return "", 0, 0, err
	}
	if opts.MaxTokens > 0 {
		filtered = truncateToTokenBudget(filtered, opts.MaxTokens)
	}
	if opts.MaxLines > 0 {
		filtered = truncateLines(filtered, opts.MaxLines)
	}
	if opts.LineNumbers {
		filtered = addLineNumbers(filtered)
	}
	result := buildResult{
		Output:         filtered,
		OriginalTokens: filter.EstimateTokens(content),
		FinalTokens:    filter.EstimateTokens(filtered),
	}
	if key, ok := cacheKey(filePath, content, opts); ok {
		renderCache.Set(key, result)
	}
	return result.Output, result.OriginalTokens, result.FinalTokens, nil
}

// ApplyMode applies the selected read mode before line or token budgeting.
func ApplyMode(filePath, content string, opts Options) (string, error) {
	if opts.Mode == "" {
		return applyLegacyReadLevel(content, opts.Level)
	}
	if opts.Mode == "graph" {
		return applyGraphRead(filePath, content, opts)
	}

	mode, err := ResolveMode(opts.Mode, filePath, content, opts.StartLine, opts.EndLine)
	if err != nil {
		return "", err
	}

	if mode == filter.ReadDiff {
		if filePath == "stdin" {
			return "", fmt.Errorf("delta mode requires a file path")
		}
		return applyDeltaRead(filePath, content, opts.SaveSnapshot)
	}

	return filter.ReadContent(content, filter.ReadOptions{
		Mode:      mode,
		StartLine: opts.StartLine,
		EndLine:   opts.EndLine,
		MaxTokens: opts.MaxTokens,
	}), nil
}

// ResolveMode translates user input into a read mode.
func ResolveMode(mode, filePath, content string, startLine, endLine int) (filter.ReadMode, error) {
	if mode == "auto" {
		return DetectAutoMode(filePath, content, startLine, endLine), nil
	}
	if mode == "delta" {
		return filter.ReadDiff, nil
	}

	switch filter.ReadMode(mode) {
	case filter.ReadFull, filter.ReadMap, filter.ReadSignatures, filter.ReadAggressive, filter.ReadEntropy, filter.ReadLines, filter.ReadDiff:
		return filter.ReadMode(mode), nil
	default:
		return "", fmt.Errorf("invalid read mode: %s (use: auto, full, map, signatures, aggressive, entropy, lines, delta, graph)", mode)
	}
}

// DetectAutoMode picks a reasonable default context mode for the file contents.
func DetectAutoMode(filePath, content string, startLine, endLine int) filter.ReadMode {
	if startLine > 0 || endLine > 0 {
		return filter.ReadLines
	}
	if strings.HasSuffix(filePath, ".diff") || strings.HasSuffix(filePath, ".patch") {
		return filter.ReadDiff
	}

	lang := detectLanguage(filePath, content)
	lines := strings.Count(content, "\n") + 1
	switch {
	case lines > 400:
		return filter.ReadMap
	case lines > 160 && lang != filter.LangUnknown:
		return filter.ReadSignatures
	case lines > 80:
		return filter.ReadAggressive
	default:
		return filter.ReadFull
	}
}

func applyLegacyReadLevel(content, level string) (string, error) {
	mode := filter.Mode(level)
	if mode != filter.ModeMinimal && mode != filter.ModeAggressive && level != "none" {
		return "", fmt.Errorf("invalid filter level: %s (use: none, minimal, aggressive)", level)
	}
	if level == "none" {
		return content, nil
	}
	engine := filter.NewEngine(mode)
	filtered, _ := engine.Process(content)
	return filtered, nil
}

func applyDeltaRead(filePath, content string, saveSnapshot bool) (string, error) {
	store, err := Load(DefaultStorePath())
	if err != nil {
		return "", fmt.Errorf("failed to load read snapshots: %w", err)
	}

	var output string
	if snap, ok := store.Get(filePath); ok {
		delta := filter.ComputeDelta(snap.Content, content)
		output = filter.FormatDelta(delta)
	} else {
		output = "No previous snapshot found. Returning current file in auto mode.\n" +
			filter.ReadContent(content, filter.ReadOptions{Mode: DetectAutoMode(filePath, content, 0, 0)})
	}

	if saveSnapshot {
		store.Put(filePath, content)
		if err := store.Save(DefaultStorePath()); err != nil {
			return "", fmt.Errorf("failed to save read snapshot: %w", err)
		}
	}

	return output, nil
}

func applyGraphRead(filePath, content string, opts Options) (string, error) {
	if filePath == "stdin" {
		return "", fmt.Errorf("graph mode requires a file path")
	}

	rootDir := detectProjectRoot(filePath)
	relPath, err := filepath.Rel(rootDir, filePath)
	if err != nil {
		relPath = filepath.Base(filePath)
	}

	g := graph.NewProjectGraph(rootDir)
	if err := g.Analyze(); err != nil {
		return "", fmt.Errorf("failed to analyze project graph: %w", err)
	}

	targetMode := DetectAutoMode(filePath, content, opts.StartLine, opts.EndLine)
	target := filter.ReadContent(content, filter.ReadOptions{
		Mode:      targetMode,
		StartLine: opts.StartLine,
		EndLine:   opts.EndLine,
		MaxTokens: opts.MaxTokens,
	})

	var out strings.Builder
	out.WriteString("# Target File\n")
	out.WriteString(relPath)
	out.WriteString("\n\n")
	out.WriteString(target)

	relatedLimit := opts.RelatedFiles
	if relatedLimit <= 0 {
		relatedLimit = 3
	}
	relatedFiles := g.FindRelatedFiles(relPath, relatedLimit)
	if len(relatedFiles) == 0 {
		return out.String(), nil
	}

	out.WriteString("\n\n# Related Files\n")
	for _, related := range relatedFiles {
		relatedPath := filepath.Join(rootDir, related)
		data, err := os.ReadFile(relatedPath)
		if err != nil {
			continue
		}

		relatedContent := string(data)
		mode := DetectAutoMode(relatedPath, relatedContent, 0, 0)
		if mode == filter.ReadFull {
			mode = filter.ReadMap
		}
		snippet := filter.ReadContent(relatedContent, filter.ReadOptions{Mode: mode})
		if strings.TrimSpace(snippet) == "" {
			continue
		}

		out.WriteString("\n## ")
		out.WriteString(related)
		out.WriteString("\n")
		out.WriteString(snippet)
		out.WriteString("\n")
	}

	return strings.TrimRight(out.String(), "\n"), nil
}

func detectProjectRoot(filePath string) string {
	dir := filepath.Dir(filePath)
	markers := []string{".git", "go.mod", "package.json", "pyproject.toml", "Cargo.toml"}

	for {
		for _, marker := range markers {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return dir
		}
		dir = parent
	}
}

// CacheStats returns smart-read render cache statistics.
func CacheStats() cache.LRUStats {
	return renderCache.Stats()
}

func cacheKey(filePath, content string, opts Options) (string, bool) {
	if filePath == "stdin" {
		return "", false
	}
	var b strings.Builder
	b.WriteString(normalizePath(filePath))
	b.WriteString("|")
	b.WriteString(cache.ComputeFingerprint(content))
	b.WriteString("|")
	b.WriteString(strings.ToLower(opts.Level))
	b.WriteString("|")
	b.WriteString(strings.ToLower(opts.Mode))
	b.WriteString("|")
	b.WriteString(strconv.Itoa(opts.MaxLines))
	b.WriteString("|")
	b.WriteString(strconv.Itoa(opts.MaxTokens))
	b.WriteString("|")
	b.WriteString(strconv.Itoa(opts.StartLine))
	b.WriteString("|")
	b.WriteString(strconv.Itoa(opts.EndLine))
	b.WriteString("|")
	b.WriteString(strconv.Itoa(opts.RelatedFiles))
	if opts.LineNumbers {
		b.WriteString("|ln")
	}
	if opts.SaveSnapshot {
		b.WriteString("|snap")
	}
	return b.String(), true
}

func truncateLines(content string, maxLines int) string {
	lines := strings.Split(content, "\n")
	if len(lines) <= maxLines {
		return content
	}

	keepStart := maxLines / 2
	keepEnd := maxLines / 4

	var result []string
	result = append(result, lines[:keepStart]...)
	result = append(result, fmt.Sprintf("// ... %d lines omitted ...", len(lines)-keepStart-keepEnd))
	result = append(result, lines[len(lines)-keepEnd:]...)
	return strings.Join(result, "\n")
}

func addLineNumbers(content string) string {
	lines := strings.Split(content, "\n")
	width := len(fmt.Sprintf("%d", len(lines)))

	var result strings.Builder
	for i, line := range lines {
		result.WriteString(fmt.Sprintf("%*d │ %s\n", width, i+1, line))
	}
	return result.String()
}

func truncateToTokenBudget(content string, maxTokens int) string {
	lines := strings.Split(content, "\n")
	if filter.EstimateTokens(content) <= maxTokens {
		return content
	}

	var kept []string
	for _, line := range lines {
		candidate := append(kept, line)
		if filter.EstimateTokens(strings.Join(candidate, "\n")) > maxTokens {
			break
		}
		kept = candidate
	}

	if len(kept) == 0 && len(lines) > 0 {
		return lines[0]
	}
	if len(kept) < len(lines) {
		kept = append(kept, fmt.Sprintf("... truncated to %d tokens ...", maxTokens))
	}
	return strings.Join(kept, "\n")
}

func detectLanguage(path string, content string) filter.Language {
	lang := detectLanguageFromExtension(path)
	if lang != filter.LangUnknown {
		return lang
	}
	return filter.DetectLanguageFromInput(content)
}

func detectLanguageFromExtension(path string) filter.Language {
	ext := strings.ToLower(filepathExt(path))
	switch ext {
	case ".go":
		return filter.LangGo
	case ".rs":
		return filter.LangRust
	case ".py", ".pyw":
		return filter.LangPython
	case ".js", ".mjs", ".cjs":
		return filter.LangJavaScript
	case ".ts", ".tsx":
		return filter.LangTypeScript
	case ".java":
		return filter.LangJava
	case ".c", ".h":
		return filter.LangC
	case ".cpp", ".cc", ".cxx", ".hpp", ".hh":
		return filter.LangCpp
	case ".rb":
		return filter.LangRuby
	case ".sh", ".bash", ".zsh":
		return filter.LangShell
	case ".sql":
		return filter.LangSQL
	default:
		return filter.LangUnknown
	}
}

func filepathExt(path string) string {
	idx := strings.LastIndex(path, ".")
	if idx < 0 {
		return ""
	}
	return path[idx:]
}
