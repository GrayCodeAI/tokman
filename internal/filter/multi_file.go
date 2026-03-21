package filter

import (
	"sort"
	"strings"
)

// MultiFileFilter optimizes output across multiple related files/outputs.
// It identifies relationships between files, deduplicates common content,
// and creates unified summaries for better LLM context.
//
// Use case: When an agent works with multiple related files simultaneously
// (e.g., a module with multiple source files), this filter creates a
// cohesive view that preserves relationships while removing redundancy.
type MultiFileFilter struct {
	// Maximum size for combined output before compression
	maxCombinedSize int
	// Whether to preserve file boundaries in output
	preserveBoundaries bool
	// Minimum similarity threshold for deduplication
	similarityThreshold float64
}

// MultiFileConfig holds configuration for multi-file optimization
type MultiFileConfig struct {
	MaxCombinedSize     int
	PreserveBoundaries  bool
	SimilarityThreshold float64
}

// NewMultiFileFilter creates a new multi-file optimization filter
func NewMultiFileFilter(cfg MultiFileConfig) *MultiFileFilter {
	f := &MultiFileFilter{
		maxCombinedSize:     cfg.MaxCombinedSize,
		preserveBoundaries:  cfg.PreserveBoundaries,
		similarityThreshold: cfg.SimilarityThreshold,
	}

	// Set defaults
	if f.maxCombinedSize == 0 {
		f.maxCombinedSize = 50000 // ~12.5K tokens
	}
	if f.similarityThreshold == 0 {
		f.similarityThreshold = 0.8 // 80% similarity = duplicate
	}

	return f
}

// Name returns the filter name
func (f *MultiFileFilter) Name() string {
	return "multi_file"
}

// Apply applies multi-file optimization
func (f *MultiFileFilter) Apply(input string, mode Mode) (string, int) {
	// Parse file markers in input
	files := f.parseFiles(input)

	// Single file - pass through
	if len(files) <= 1 {
		return input, 0
	}

	// Analyze relationships
	relations := f.analyzeRelationships(files)

	// Deduplicate common content
	deduped := f.deduplicate(files, relations)

	// Create unified output
	output := f.createUnifiedOutput(deduped, mode)

	tokensSaved := EstimateTokens(input) - EstimateTokens(output)
	if tokensSaved < 0 {
		tokensSaved = 0
	}

	return output, tokensSaved
}

// fileInfo represents parsed file content
type fileInfo struct {
	name      string
	content   string
	startPos  int
	endPos    int
	imports   []string
	exports   []string
	types     []string
	functions []string
}

// parseFiles extracts individual files from combined input
func (f *MultiFileFilter) parseFiles(input string) []fileInfo {
	var files []fileInfo

	// Common file markers
	markers := []string{
		"=== File: ",
		"--- ",
		"diff --git a/",
		"// File: ",
		"# File: ",
		"/* File: ",
	}

	lines := strings.Split(input, "\n")
	var currentFile *fileInfo
	var currentContent []string

	for i, line := range lines {
		isFileStart := false
		var fileName string

		for _, marker := range markers {
			if strings.HasPrefix(line, marker) {
				isFileStart = true
				fileName = strings.TrimPrefix(line, marker)
				fileName = strings.TrimSpace(fileName)
				break
			}
		}

		// Also detect file paths in diff format
		if strings.HasPrefix(line, "+++ b/") || strings.HasPrefix(line, "--- a/") {
			isFileStart = true
			fileName = strings.TrimPrefix(line, "+++ b/")
			fileName = strings.TrimPrefix(fileName, "--- a/")
			fileName = strings.TrimSpace(fileName)
		}

		if isFileStart && fileName != "" {
			// Save previous file
			if currentFile != nil && len(currentContent) > 0 {
				currentFile.content = strings.Join(currentContent, "\n")
				currentFile.endPos = i - 1
				f.extractMetadata(currentFile)
				files = append(files, *currentFile)
			}

			// Start new file
			currentFile = &fileInfo{
				name:     fileName,
				startPos: i,
			}
			currentContent = nil
		} else if currentFile != nil {
			currentContent = append(currentContent, line)
		}
	}

	// Save last file
	if currentFile != nil && len(currentContent) > 0 {
		currentFile.content = strings.Join(currentContent, "\n")
		currentFile.endPos = len(lines) - 1
		f.extractMetadata(currentFile)
		files = append(files, *currentFile)
	}

	// If no file markers found, treat entire input as one file
	if len(files) == 0 {
		files = append(files, fileInfo{
			name:    "output",
			content: input,
		})
	}

	return files
}

// extractMetadata extracts imports, exports, types, functions from file content
func (f *MultiFileFilter) extractMetadata(file *fileInfo) {
	content := file.content
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Imports
		if strings.HasPrefix(trimmed, "import ") ||
			strings.HasPrefix(trimmed, "use ") ||
			strings.HasPrefix(trimmed, "require(") ||
			strings.HasPrefix(trimmed, "from ") {
			file.imports = append(file.imports, trimmed)
		}

		// Exports (public functions, types)
		if strings.HasPrefix(trimmed, "export ") ||
			strings.HasPrefix(trimmed, "pub fn ") ||
			strings.HasPrefix(trimmed, "pub struct ") ||
			strings.HasPrefix(trimmed, "func (") && strings.Contains(trimmed, ") ") {
			file.exports = append(file.exports, trimmed)
		}

		// Types
		if strings.HasPrefix(trimmed, "type ") ||
			strings.HasPrefix(trimmed, "struct ") ||
			strings.HasPrefix(trimmed, "interface ") ||
			strings.HasPrefix(trimmed, "class ") {
			file.types = append(file.types, trimmed)
		}

		// Functions
		if strings.HasPrefix(trimmed, "func ") ||
			strings.HasPrefix(trimmed, "fn ") ||
			strings.HasPrefix(trimmed, "def ") ||
			strings.HasPrefix(trimmed, "function ") {
			file.functions = append(file.functions, trimmed)
		}
	}

	// Limit arrays
	if len(file.imports) > 20 {
		file.imports = file.imports[:20]
	}
	if len(file.exports) > 20 {
		file.exports = file.exports[:20]
	}
	if len(file.types) > 10 {
		file.types = file.types[:10]
	}
	if len(file.functions) > 15 {
		file.functions = file.functions[:15]
	}
}

// fileRelation represents a relationship between two files
type fileRelation struct {
	file1    string
	file2    string
	relation string // "imports", "exports-to", "similar", "same-module"
	strength float64
}

// analyzeRelationships finds connections between files
func (f *MultiFileFilter) analyzeRelationships(files []fileInfo) []fileRelation {
	var relations []fileRelation

	for i, file1 := range files {
		for j, file2 := range files {
			if i >= j {
				continue
			}

			// Check for import relationships
			if f.hasImportRelation(file1, file2) {
				relations = append(relations, fileRelation{
					file1:    file1.name,
					file2:    file2.name,
					relation: "imports",
					strength: 0.8,
				})
			}

			// Check for same module
			if f.sameModule(file1.name, file2.name) {
				relations = append(relations, fileRelation{
					file1:    file1.name,
					file2:    file2.name,
					relation: "same-module",
					strength: 0.6,
				})
			}

			// Check for content similarity
			similarity := f.calculateSimilarity(file1.content, file2.content)
			if similarity >= f.similarityThreshold {
				relations = append(relations, fileRelation{
					file1:    file1.name,
					file2:    file2.name,
					relation: "similar",
					strength: similarity,
				})
			}
		}
	}

	// Sort by strength
	sort.Slice(relations, func(i, j int) bool {
		return relations[i].strength > relations[j].strength
	})

	return relations
}

// hasImportRelation checks if file1 imports from file2
func (f *MultiFileFilter) hasImportRelation(file1, file2 fileInfo) bool {
	file2Base := file2.name
	if idx := strings.LastIndex(file2Base, "/"); idx >= 0 {
		file2Base = file2Base[idx+1:]
	}
	if idx := strings.LastIndex(file2Base, "."); idx >= 0 {
		file2Base = file2Base[:idx]
	}

	for _, imp := range file1.imports {
		if strings.Contains(imp, file2Base) {
			return true
		}
	}

	return false
}

// sameModule checks if two files are in the same module/directory
func (f *MultiFileFilter) sameModule(file1, file2 string) bool {
	dir1 := ""
	dir2 := ""

	if idx := strings.LastIndex(file1, "/"); idx >= 0 {
		dir1 = file1[:idx]
	}
	if idx := strings.LastIndex(file2, "/"); idx >= 0 {
		dir2 = file2[:idx]
	}

	// Both in root directory (no path separator) counts as same module
	return dir1 == dir2
}

// calculateSimilarity computes content similarity between two strings
func (f *MultiFileFilter) calculateSimilarity(content1, content2 string) float64 {
	// Simple token-based similarity
	tokens1 := f.tokenize(content1)
	tokens2 := f.tokenize(content2)

	if len(tokens1) == 0 || len(tokens2) == 0 {
		return 0.0
	}

	// Count common tokens
	common := 0
	for token := range tokens1 {
		if tokens2[token] {
			common++
		}
	}

	// Jaccard similarity
	total := len(tokens1) + len(tokens2) - common
	if total == 0 {
		return 0.0
	}

	return float64(common) / float64(total)
}

// tokenize splits content into tokens for comparison
func (f *MultiFileFilter) tokenize(content string) map[string]bool {
	tokens := make(map[string]bool)
	words := strings.Fields(content)

	for _, word := range words {
		// Normalize
		word = strings.ToLower(strings.TrimSpace(word))
		if len(word) > 2 { // Skip short tokens
			tokens[word] = true
		}
	}

	return tokens
}

// deduplicatedFile represents a file after deduplication
type deduplicatedFile struct {
	name    string
	content string
	shared  []string // Content shared with other files
	unique  []string // Content unique to this file
	imports []string
	exports []string
}

// deduplicate removes common content from related files
func (f *MultiFileFilter) deduplicate(files []fileInfo, relations []fileRelation) []deduplicatedFile {
	var result []deduplicatedFile

	// Find shared imports across files
	sharedImports := f.findSharedImports(files)

	for _, file := range files {
		deduped := deduplicatedFile{
			name:    file.name,
			imports: file.imports,
			exports: file.exports,
		}

		// Remove shared imports from individual files
		deduped.imports = f.removeShared(file.imports, sharedImports)

		// For similar files, extract unique content
		isSimilar := false
		for _, rel := range relations {
			if (rel.file1 == file.name || rel.file2 == file.name) && rel.relation == "similar" {
				isSimilar = true
				break
			}
		}

		if isSimilar {
			deduped.content = f.extractUniqueContent(file, files, relations)
		} else {
			deduped.content = file.content
		}

		result = append(result, deduped)
	}

	return result
}

// findSharedImports finds imports common to multiple files
func (f *MultiFileFilter) findSharedImports(files []fileInfo) []string {
	importCounts := make(map[string]int)

	for _, file := range files {
		seen := make(map[string]bool)
		for _, imp := range file.imports {
			// Normalize import
			normalized := strings.ToLower(strings.TrimSpace(imp))
			if !seen[normalized] {
				importCounts[normalized]++
				seen[normalized] = true
			}
		}
	}

	var shared []string
	for imp, count := range importCounts {
		if count > 1 { // Shared by at least 2 files
			shared = append(shared, imp)
		}
	}

	return shared
}

// removeShared removes shared items from a list
func (f *MultiFileFilter) removeShared(items, shared []string) []string {
	sharedSet := make(map[string]bool)
	for _, s := range shared {
		sharedSet[strings.ToLower(s)] = true
	}

	var result []string
	for _, item := range items {
		if !sharedSet[strings.ToLower(item)] {
			result = append(result, item)
		}
	}

	return result
}

// extractUniqueContent extracts content unique to a file
func (f *MultiFileFilter) extractUniqueContent(file fileInfo, allFiles []fileInfo, relations []fileRelation) string {
	// Find similar files
	var similarFiles []fileInfo
	for _, rel := range relations {
		if rel.relation == "similar" {
			var otherName string
			if rel.file1 == file.name {
				otherName = rel.file2
			} else if rel.file2 == file.name {
				otherName = rel.file1
			}

			if otherName != "" {
				for _, other := range allFiles {
					if other.name == otherName {
						similarFiles = append(similarFiles, other)
					}
				}
			}
		}
	}

	if len(similarFiles) == 0 {
		return file.content
	}

	// Extract lines unique to this file
	lines := strings.Split(file.content, "\n")
	var uniqueLines []string

	for _, line := range lines {
		isUnique := true
		for _, similar := range similarFiles {
			if strings.Contains(similar.content, line) && len(line) > 10 {
				isUnique = false
				break
			}
		}
		if isUnique || len(line) <= 10 {
			uniqueLines = append(uniqueLines, line)
		}
	}

	return strings.Join(uniqueLines, "\n")
}

// createUnifiedOutput creates the final combined output
func (f *MultiFileFilter) createUnifiedOutput(files []deduplicatedFile, mode Mode) string {
	var output strings.Builder

	// Add shared imports section first
	sharedImports := f.findSharedImportsFromDeduped(files)
	if len(sharedImports) > 0 && f.preserveBoundaries {
		output.WriteString("=== Shared Imports ===\n")
		for _, imp := range sharedImports {
			output.WriteString(imp + "\n")
		}
		output.WriteString("\n")
	}

	// Add each file
	for i, file := range files {
		if f.preserveBoundaries {
			output.WriteString("=== File: " + file.name + " ===\n")
		}

		// In aggressive mode, only show signatures
		if mode == ModeAggressive {
			if len(file.exports) > 0 {
				output.WriteString("Exports:\n")
				for _, exp := range file.exports {
					output.WriteString("  " + exp + "\n")
				}
			}
			if len(file.imports) > 0 {
				output.WriteString("Imports:\n")
				for _, imp := range file.imports {
					output.WriteString("  " + imp + "\n")
				}
			}
		} else {
			// Show full content
			output.WriteString(file.content)
			if !strings.HasSuffix(file.content, "\n") {
				output.WriteString("\n")
			}
		}

		if f.preserveBoundaries && i < len(files)-1 {
			output.WriteString("\n")
		}
	}

	return output.String()
}

// findSharedImportsFromDeduped finds shared imports from deduplicated files
func (f *MultiFileFilter) findSharedImportsFromDeduped(files []deduplicatedFile) []string {
	importCounts := make(map[string]int)

	for _, file := range files {
		seen := make(map[string]bool)
		for _, imp := range file.imports {
			normalized := strings.ToLower(strings.TrimSpace(imp))
			if !seen[normalized] && normalized != "" {
				importCounts[normalized]++
				seen[normalized] = true
			}
		}
	}

	var shared []string
	for imp, count := range importCounts {
		if count > 1 {
			shared = append(shared, imp)
		}
	}

	sort.Strings(shared)
	return shared
}

// SetMaxCombinedSize configures the maximum combined output size
func (f *MultiFileFilter) SetMaxCombinedSize(size int) {
	f.maxCombinedSize = size
}

// SetPreserveBoundaries configures whether to keep file markers
func (f *MultiFileFilter) SetPreserveBoundaries(preserve bool) {
	f.preserveBoundaries = preserve
}

// SetSimilarityThreshold configures the deduplication threshold
func (f *MultiFileFilter) SetSimilarityThreshold(threshold float64) {
	f.similarityThreshold = threshold
}
