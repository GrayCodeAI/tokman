package filter

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

var paragraphSplitRe = regexp.MustCompile(`\n\s*\n`)

// SemanticCacheFilter implements SemantiCache-style clustered merging.
// Research Source: "SemantiCache: Efficient KV Cache Compression via Semantic
// Chunking and Clustered Merging" (Mar 2026)
// Key Innovation: Group tokens into semantic clusters, then merge each cluster
// into a "semantic core" using proportional attention rebalancing.
// Results: 2.61x decode speedup, preserves semantic integrity.
//
// This compresses by finding semantically similar sentences/paragraphs and
// merging them into representative cores, reducing redundancy while preserving
// unique information.
type SemanticCacheFilter struct {
	config SemanticCacheConfig
}

// SemanticCacheConfig holds configuration for semantic caching
type SemanticCacheConfig struct {
	// Enabled controls whether the filter is active
	Enabled bool

	// SimilarityThreshold for clustering (0-1). Higher = stricter matching
	SimilarityThreshold float64

	// MinClusterSize minimum items before merging
	MinClusterSize int

	// MaxCores maximum semantic cores to keep
	MaxCores int

	// MinContentLength minimum chars to apply
	MinContentLength int
}

// DefaultSemanticCacheConfig returns default configuration
func DefaultSemanticCacheConfig() SemanticCacheConfig {
	return SemanticCacheConfig{
		Enabled:             true,
		SimilarityThreshold: 0.7,
		MinClusterSize:      3,
		MaxCores:            20,
		MinContentLength:    500,
	}
}

// NewSemanticCacheFilter creates a new semantic cache filter
func NewSemanticCacheFilter() *SemanticCacheFilter {
	return &SemanticCacheFilter{
		config: DefaultSemanticCacheConfig(),
	}
}

// Name returns the filter name
func (f *SemanticCacheFilter) Name() string {
	return "semantic_cache"
}

// semanticCluster holds a group of similar items
type semanticCluster struct {
	items     []string
	core      string // Representative core
	frequency int
}

// Apply applies semantic cache compression
func (f *SemanticCacheFilter) Apply(input string, mode Mode) (string, int) {
	if !f.config.Enabled || mode == ModeNone {
		return input, 0
	}

	if len(input) < f.config.MinContentLength {
		return input, 0
	}

	originalTokens := core.EstimateTokens(input)

	// Split into semantic units (paragraphs or lines)
	units := f.splitIntoUnits(input)
	if len(units) < f.config.MinClusterSize*2 {
		return input, 0
	}

	// Build semantic clusters using greedy seed-based clustering
	clusters := f.greedySeedCluster(units)

	// Merge clusters into semantic cores
	output := f.mergeToCores(clusters, mode)

	finalTokens := core.EstimateTokens(output)
	saved := originalTokens - finalTokens
	if saved < 5 {
		return input, 0
	}

	return output, saved
}

// splitIntoUnits splits content into semantic units
func (f *SemanticCacheFilter) splitIntoUnits(input string) []string {
	// Try paragraph splitting first
	paragraphs := splitParagraphs(input)
	if len(paragraphs) >= f.config.MinClusterSize*2 {
		return paragraphs
	}

	// Fall back to line splitting
	lines := strings.Split(input, "\n")
	var units []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			units = append(units, trimmed)
		}
	}
	return units
}

// greedySeedCluster implements Greedy Seed-Based Clustering (GSC).
// Precomputes the full pairwise similarity matrix once (O(n²)) to avoid
// redundant similarity computation during seed selection and assignment.
func (f *SemanticCacheFilter) greedySeedCluster(units []string) []semanticCluster {
	n := len(units)
	assigned := make([]bool, n)
	var clusters []semanticCluster

	// Precompute pairwise similarity matrix: O(n²) once.
	sim := make([][]float64, n)
	for i := 0; i < n; i++ {
		sim[i] = make([]float64, n)
	}
	for i := 0; i < n; i++ {
		sim[i][i] = 1.0
		for j := i + 1; j < n; j++ {
			s := f.computeSimilarity(units[i], units[j])
			sim[i][j] = s
			sim[j][i] = s
		}
	}

	// Precompute representativeness scores (sum of similarity to all others).
	repScore := make([]float64, n)
	for i := 0; i < n; i++ {
		total := 0.0
		for j := 0; j < n; j++ {
			if j != i {
				total += sim[i][j]
			}
		}
		repScore[i] = total
	}

	for {
		// Find unassigned item with highest representativeness: O(n).
		seedIdx := -1
		bestScore := -1.0
		for i := 0; i < n; i++ {
			if !assigned[i] && repScore[i] > bestScore {
				bestScore = repScore[i]
				seedIdx = i
			}
		}

		if seedIdx < 0 || len(clusters) >= f.config.MaxCores {
			break
		}

		// Create cluster with seed.
		cluster := semanticCluster{
			items:     []string{units[seedIdx]},
			core:      units[seedIdx],
			frequency: 1,
		}
		assigned[seedIdx] = true

		// Assign similar items to this cluster: O(n) using precomputed sim.
		for i := 0; i < n; i++ {
			if assigned[i] {
				continue
			}
			if sim[seedIdx][i] >= f.config.SimilarityThreshold {
				cluster.items = append(cluster.items, units[i])
				cluster.frequency++
				assigned[i] = true
				// Remove this item's contribution from remaining repScores.
				for k := 0; k < n; k++ {
					if !assigned[k] {
						repScore[k] -= sim[k][i]
					}
				}
			}
		}
		// Remove seed's contribution from remaining repScores.
		for k := 0; k < n; k++ {
			if !assigned[k] {
				repScore[k] -= sim[k][seedIdx]
			}
		}

		clusters = append(clusters, cluster)
	}

	// Add any remaining unassigned items as single-item clusters.
	for i := 0; i < n; i++ {
		if !assigned[i] {
			clusters = append(clusters, semanticCluster{
				items:     []string{units[i]},
				core:      units[i],
				frequency: 1,
			})
		}
	}

	return clusters
}

// computeSimilarity computes semantic similarity between two text units.
// Uses Jaccard similarity on word sets with n-gram overlap bonus.
func (f *SemanticCacheFilter) computeSimilarity(a, b string) float64 {
	if a == b {
		return 1.0
	}

	wordsA := extractWordSet(a)
	wordsB := extractWordSet(b)

	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0
	}

	// Jaccard similarity
	intersection := 0
	for word := range wordsA {
		if wordsB[word] {
			intersection++
		}
	}

	union := len(wordsA) + len(wordsB) - intersection
	if union == 0 {
		return 0
	}

	jaccard := float64(intersection) / float64(union)

	// N-gram overlap bonus
	ngramSim := f.computeNgramSimilarity(a, b)

	return jaccard*0.7 + ngramSim*0.3
}

// computeNgramSimilarity computes character n-gram overlap
func (f *SemanticCacheFilter) computeNgramSimilarity(a, b string) float64 {
	ngramsA := extractCharNgrams(strings.ToLower(a), 3)
	ngramsB := extractCharNgrams(strings.ToLower(b), 3)

	if len(ngramsA) == 0 || len(ngramsB) == 0 {
		return 0
	}

	intersection := 0
	for ng := range ngramsA {
		if ngramsB[ng] {
			intersection++
		}
	}

	union := len(ngramsA) + len(ngramsB) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

// mergeToCores merges clusters into semantic cores
func (f *SemanticCacheFilter) mergeToCores(clusters []semanticCluster, mode Mode) string {
	var result strings.Builder

	for _, cluster := range clusters {
		if cluster.frequency > 1 {
			// Multi-item cluster: show representative + count
			result.WriteString(cluster.core)
			if cluster.frequency > 2 {
				result.WriteString(" [×")
				result.WriteString(strconv.Itoa(cluster.frequency))
				result.WriteString("]")
			}
		} else {
			// Single item: keep as-is
			result.WriteString(cluster.core)
		}
		result.WriteString("\n")
	}

	return strings.TrimSpace(result.String())
}

// Helper functions
func extractWordSet(text string) map[string]bool {
	words := make(map[string]bool)
	for _, w := range strings.Fields(strings.ToLower(text)) {
		cleaned := strings.Trim(w, ".,;:!?\"'()[]{}")
		if len(cleaned) > 1 {
			words[cleaned] = true
		}
	}
	return words
}

func extractCharNgrams(text string, n int) map[string]bool {
	ngrams := make(map[string]bool)
	runes := []rune(text)
	for i := 0; i <= len(runes)-n; i++ {
		ngrams[string(runes[i:i+n])] = true
	}
	return ngrams
}

func splitParagraphs(text string) []string {
	parts := paragraphSplitRe.Split(text, -1)
	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
