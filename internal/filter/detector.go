package filter

// ContentAnalyzer provides content type detection for conditional layer execution.
// T86: Skip layers that don't apply to current content type.
type ContentAnalyzer struct {
	selector *AdaptiveLayerSelector
}

// NewContentAnalyzer creates a content analyzer.
func NewContentAnalyzer() *ContentAnalyzer {
	return &ContentAnalyzer{
		selector: NewAdaptiveLayerSelector(),
	}
}

// Analyze detects the content type of input.
func (a *ContentAnalyzer) Analyze(input string) ContentType {
	return a.selector.AnalyzeContent(input)
}

// ShouldSkipLayer returns true if a layer should be skipped for the given content type.
// T86: Conditional layer execution based on content.
func ShouldSkipLayer(layerName string, contentType ContentType) bool {
	switch layerName {
	case "ast_preserve":
		// AST preservation only useful for code
		return contentType != ContentTypeCode
	case "ngram":
		// N-gram abbreviation works best on logs and prose
		return contentType == ContentTypeCode
	case "gist":
		// Gist compression works best on long prose
		return contentType == ContentTypeCode || contentType == ContentTypeGitOutput
	}
	return false
}
