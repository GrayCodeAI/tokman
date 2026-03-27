package filter_test

import (
	"strings"
	"testing"

	"github.com/GrayCodeAI/tokman/internal/filter"
)

// Representative test corpora.

var corpusCode = strings.Repeat(`package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", handler)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %s!", r.URL.Path[1:])
}
`, 3)

var corpusLogs = strings.Repeat(`2025-03-27T10:00:00Z INFO  Starting service version=1.2.3
2025-03-27T10:00:01Z DEBUG Connecting to database host=localhost port=5432
2025-03-27T10:00:02Z WARN  Retry attempt 1 of 3 err="connection refused"
2025-03-27T10:00:03Z ERROR Failed to process request id=abc123 err="timeout after 30s"
2025-03-27T10:00:04Z INFO  Request completed status=200 latency=142ms
`, 10)

var corpusJSON = strings.Repeat(`{"id":1,"name":"widget","price":9.99,"tags":["sale","new"],"meta":{"created":"2025-01-01","updated":"2025-03-27"}}
`, 8)

var corpusMarkdown = strings.Repeat(`## Overview

This module handles **authentication** and session management.

### Features

- OAuth2 support
- JWT tokens
- Rate limiting (100 req/s per client)

> Note: tokens expire after 24 hours.

`, 5)

// BenchmarkAllFilters runs each common filter against a 1 KB corpus.
func BenchmarkAllFilters(b *testing.B) {
	// Build a ~1 KB input mixing all corpus types.
	corpus1KB := (corpusCode + corpusLogs + corpusJSON + corpusMarkdown)
	if len(corpus1KB) > 1024 {
		corpus1KB = corpus1KB[:1024]
	}

	type filterCase struct {
		name string
		fn   func(string, filter.Mode) (string, int)
	}

	cases := []filterCase{
		{"ANSIFilter", filter.NewANSIFilter().Apply},
		{"SemanticFilter", filter.NewSemanticFilter().Apply},
		{"PositionAwareFilter", filter.NewPositionAwareFilter().Apply},
		{"HierarchicalFilter", filter.NewHierarchicalFilter().Apply},
		{"EntropyFilter", filter.NewEntropyFilter().Apply},
		{"PerplexityFilter", filter.NewPerplexityFilter().Apply},
		{"ASTPreserveFilter", filter.NewASTPreserveFilter().Apply},
		{"ContrastiveFilter", filter.NewContrastiveFilter("debug error").Apply},
		{"EvaluatorHeadsFilter", filter.NewEvaluatorHeadsFilter().Apply},
		{"GistFilter", filter.NewGistFilter().Apply},
		{"AttributionFilter", filter.NewAttributionFilter().Apply},
		{"H2OFilter", filter.NewH2OFilter().Apply},
		{"AttentionSinkFilter", filter.NewAttentionSinkFilter().Apply},
		{"SketchStoreFilter", filter.NewSketchStoreFilter().Apply},
		{"DynamicRatioFilter", filter.NewDynamicRatioFilter().Apply},
	}

	for _, tc := range cases {
		tc := tc
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				out, _ := tc.fn(corpus1KB, filter.ModeMinimal)
				_ = out
			}
		})
	}
}

// BenchmarkPipeline_Small benchmarks the full pipeline at ~100 token input.
func BenchmarkPipeline_Small(b *testing.B) {
	input := strings.Repeat("the quick brown fox jumps over the lazy dog. ", 5) // ~100 tokens
	p := filter.NewPipelineCoordinator(filter.PipelineConfig{
		Mode:            filter.ModeMinimal,
		SessionTracking: true,
		NgramEnabled:    true,
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, stats := p.Process(input)
		tokens := float64(stats.OriginalTokens)
		if tokens > 0 {
			b.ReportMetric(tokens/float64(b.Elapsed().Seconds()+1e-9)*float64(b.N), "tokens/s")
		}
		_ = out
	}
}

// BenchmarkPipeline_Medium benchmarks the full pipeline at ~1 K token input.
func BenchmarkPipeline_Medium(b *testing.B) {
	input := strings.Repeat(corpusLogs, 2) // ~1 K tokens
	p := filter.NewPipelineCoordinator(filter.PipelineConfig{
		Mode:              filter.ModeMinimal,
		SessionTracking:   true,
		NgramEnabled:      true,
		EnableEntropy:     true,
		EnablePerplexity:  true,
		EnableH2O:         true,
		EnableAttribution: true,
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, stats := p.Process(input)
		tokens := float64(stats.OriginalTokens)
		if tokens > 0 {
			b.ReportMetric(tokens/float64(b.Elapsed().Seconds()+1e-9)*float64(b.N), "tokens/s")
		}
		_ = out
	}
}

// BenchmarkPipeline_Large benchmarks the full pipeline at ~10 K token input.
func BenchmarkPipeline_Large(b *testing.B) {
	input := strings.Repeat(corpusCode+corpusLogs+corpusJSON+corpusMarkdown, 10) // ~10 K tokens
	p := filter.NewPipelineCoordinator(filter.PipelineConfig{
		Mode:                filter.ModeAggressive,
		SessionTracking:     true,
		NgramEnabled:        true,
		EnableEntropy:       true,
		EnablePerplexity:    true,
		EnableGoalDriven:    true,
		EnableAST:           true,
		EnableContrastive:   true,
		EnableH2O:           true,
		EnableAttentionSink: true,
		EnableAttribution:   true,
		EnableDynamicRatio:  true,
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, stats := p.Process(input)
		tokens := float64(stats.OriginalTokens)
		if tokens > 0 {
			b.ReportMetric(tokens/float64(b.Elapsed().Seconds()+1e-9)*float64(b.N), "tokens/s")
		}
		_ = out
	}
}

// BenchmarkParallel exercises the pipeline concurrently using b.RunParallel.
func BenchmarkParallel(b *testing.B) {
	input := corpusCode + corpusLogs + corpusJSON + corpusMarkdown
	// Each goroutine gets its own coordinator to avoid shared state.
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		p := filter.NewPipelineCoordinator(filter.PipelineConfig{
			Mode:              filter.ModeMinimal,
			SessionTracking:   true,
			NgramEnabled:      true,
			EnableEntropy:     true,
			EnableH2O:         true,
			EnableAttribution: true,
		})
		for pb.Next() {
			out, stats := p.Process(input)
			tokens := float64(stats.OriginalTokens)
			if tokens > 0 {
				b.ReportMetric(tokens, "tokens/s")
			}
			_ = out
		}
	})
}
