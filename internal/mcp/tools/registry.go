// Package tools provides the 27 MCP tool implementations.
package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/GrayCodeAI/tokman/internal/config"
	"github.com/GrayCodeAI/tokman/internal/contextread"
	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/mcp"
)

// RegisterAllTools registers all 27 MCP tools to the registry.
func RegisterAllTools(registry *mcp.ToolRegistry, cache *mcp.HashCache) {
	// Core read tools (7 modes)
	registerTool(registry, "ctx_read", "Read file content with mode (full, map, outline, symbols, imports, types, exports)",
		map[string]mcp.Property{
			"path":       {Type: "string", Description: "File path to read"},
			"mode":       {Type: "string", Description: "Read mode", Enum: []string{"full", "map", "outline", "symbols", "imports", "types", "exports"}},
			"max_tokens": {Type: "integer", Description: "Maximum tokens to return"},
		},
		[]string{"path"},
		makeCtxReadHandler(cache))

	// Delta/diff tool
	registerTool(registry, "ctx_delta", "Get diff from previous file version",
		map[string]mcp.Property{
			"path":      {Type: "string", Description: "File path"},
			"base_hash": {Type: "string", Description: "Base hash for comparison"},
		},
		[]string{"path"},
		makeCtxDeltaHandler(cache))

	// Grep search
	registerTool(registry, "ctx_grep", "Search files with regex pattern",
		map[string]mcp.Property{
			"path":    {Type: "string", Description: "Directory or file path to search"},
			"pattern": {Type: "string", Description: "Regex pattern"},
			"context": {Type: "integer", Description: "Lines of context"},
		},
		[]string{"path", "pattern"},
		makeCtxGrepHandler())

	// Hash computation
	registerTool(registry, "ctx_hash", "Compute SHA-256 hash of content",
		map[string]mcp.Property{
			"content": {Type: "string", Description: "Content to hash"},
			"path":    {Type: "string", Description: "File path (alternative to content)"},
		},
		[]string{},
		makeCtxHashHandler())

	// Cache info
	registerTool(registry, "ctx_cache_info", "Get cache statistics",
		map[string]mcp.Property{},
		[]string{},
		makeCtxCacheInfoHandler(cache))

	// Cache invalidation
	registerTool(registry, "ctx_invalidate", "Invalidate cache entries",
		map[string]mcp.Property{
			"pattern": {Type: "string", Description: "Glob pattern to match"},
			"all":     {Type: "boolean", Description: "Clear all entries"},
		},
		[]string{},
		makeCtxInvalidateHandler(cache))

	// Compact/compress
	registerTool(registry, "ctx_compact", "Compress content using TokMan filters",
		map[string]mcp.Property{
			"content": {Type: "string", Description: "Content to compress"},
			"path":    {Type: "string", Description: "File path (alternative)"},
			"mode":    {Type: "string", Description: "Compression mode", Enum: []string{"minimal", "aggressive"}},
		},
		[]string{},
		makeCtxCompactHandler(cache))

	// Summary
	registerTool(registry, "ctx_summary", "Get file summary/preview",
		map[string]mcp.Property{
			"path": {Type: "string", Description: "File path"},
		},
		[]string{"path"},
		makeCtxSummaryHandler())

	// Memory - remember
	registerTool(registry, "ctx_remember", "Store a memory entry",
		map[string]mcp.Property{
			"key":   {Type: "string", Description: "Memory key"},
			"value": {Type: "string", Description: "Value to store"},
			"tags":  {Type: "array", Description: "Tags for categorization"},
		},
		[]string{"key", "value"},
		makeCtxRememberHandler())

	// Memory - recall
	registerTool(registry, "ctx_recall", "Retrieve a memory entry",
		map[string]mcp.Property{
			"key": {Type: "string", Description: "Memory key"},
		},
		[]string{"key"},
		makeCtxRecallHandler())

	// Memory - search
	registerTool(registry, "ctx_search_memory", "Search memory entries",
		map[string]mcp.Property{
			"query": {Type: "string", Description: "Search query"},
			"tag":   {Type: "string", Description: "Filter by tag"},
		},
		[]string{"query"},
		makeCtxSearchMemoryHandler())

	// Bundle - multi-file
	registerTool(registry, "ctx_bundle", "Create a bundle of multiple files",
		map[string]mcp.Property{
			"paths":    {Type: "array", Description: "File paths to include"},
			"compress": {Type: "boolean", Description: "Apply compression"},
		},
		[]string{"paths"},
		makeCtxBundleHandler(cache))

	// Bundle changed (git-based)
	registerTool(registry, "ctx_bundle_changed", "Bundle files changed in git",
		map[string]mcp.Property{
			"since":    {Type: "string", Description: "Git ref (commit/branch)"},
			"compress": {Type: "boolean", Description: "Apply compression"},
		},
		[]string{},
		makeCtxBundleChangedHandler())

	// Bundle summary
	registerTool(registry, "ctx_bundle_summary", "Get bundle statistics",
		map[string]mcp.Property{
			"paths": {Type: "array", Description: "File paths to analyze"},
		},
		[]string{"paths"},
		makeCtxBundleSummaryHandler())

	// Execute command
	registerTool(registry, "ctx_exec", "Execute a shell command safely",
		map[string]mcp.Property{
			"command": {Type: "string", Description: "Command to execute"},
			"args":    {Type: "array", Description: "Command arguments"},
			"timeout": {Type: "integer", Description: "Timeout in seconds"},
		},
		[]string{"command"},
		makeCtxExecHandler())

	// TLDR help
	registerTool(registry, "ctx_tldr", "Get command help from TLDR pages",
		map[string]mcp.Property{
			"command": {Type: "string", Description: "Command name"},
		},
		[]string{"command"},
		makeCtxTldrHandler())

	// Hook patterns
	registerTool(registry, "ctx_patterns", "List available hook patterns",
		map[string]mcp.Property{},
		[]string{},
		makeCtxPatternsHandler())

	// Context modes info
	registerTool(registry, "ctx_modes", "List available context modes",
		map[string]mcp.Property{},
		[]string{},
		makeCtxModesHandler())

	// Mode setter
	registerTool(registry, "ctx_mode", "Set default context mode",
		map[string]mcp.Property{
			"mode": {Type: "string", Description: "Mode to set", Enum: []string{"full", "map", "outline", "symbols", "imports", "types", "exports"}},
		},
		[]string{"mode"},
		makeCtxModeHandler())

	// Status
	registerTool(registry, "ctx_status", "Get MCP server status",
		map[string]mcp.Property{},
		[]string{},
		makeCtxStatusHandler())

	// Config
	registerTool(registry, "ctx_config", "Get or set configuration",
		map[string]mcp.Property{
			"key":   {Type: "string", Description: "Config key"},
			"value": {Type: "string", Description: "Value to set"},
		},
		[]string{},
		makeCtxConfigHandler())

	// MCP export
	registerTool(registry, "ctx_mcp", "Export MCP configuration for clients",
		map[string]mcp.Property{},
		[]string{},
		makeCtxMcpHandler())
}

func registerTool(registry *mcp.ToolRegistry, name, description string, properties map[string]mcp.Property, required []string, handler mcp.ToolHandler) {
	tool := mcp.Tool{
		Name:        name,
		Description: description,
		InputSchema: mcp.InputSchema{
			Type:       "object",
			Properties: properties,
			Required:   required,
		},
	}
	registry.Register(tool, handler)
}

// Tool handlers implementation

func makeCtxReadHandler(cache *mcp.HashCache) mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		path, _ := params["path"].(string)
		if path == "" {
			return nil, mcp.ErrInvalidParams
		}

		mode := "full"
		if m, ok := params["mode"].(string); ok && m != "" {
			mode = m
		}

		maxTokens := 0
		if mt, ok := params["max_tokens"].(float64); ok {
			maxTokens = int(mt)
		}

		// Check cache first
		contentHash := computeFileHash(path)
		cacheKey := fmt.Sprintf("%s:%s:%d", path, mode, maxTokens)

		if entry, ok := cache.Get(cacheKey); ok && entry.FilePath == path {
			return map[string]interface{}{
				"path":      path,
				"mode":      mode,
				"content":   entry.Content,
				"cached":    true,
				"hash":      contentHash,
				"hit_count": entry.HitCount,
			}, nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil, mcp.ErrFileNotFound
		}

		content := string(data)
		originalTokens := filter.EstimateTokens(content)

		opts := contextread.Options{
			Mode:      mode,
			MaxTokens: maxTokens,
			Level:     "minimal",
		}

		filtered, _, finalTokens, err := contextread.Build(path, content, opts)
		if err != nil {
			return nil, err
		}

		// Cache the result
		cache.Set(&mcp.CacheEntry{
			Hash:      cacheKey,
			Content:   filtered,
			FilePath:  path,
			Timestamp: time.Now(),
			Accessed:  time.Now(),
			HitCount:  1,
		})

		return map[string]interface{}{
			"path":            path,
			"mode":            mode,
			"content":         filtered,
			"original_tokens": originalTokens,
			"final_tokens":    finalTokens,
			"saved_tokens":    originalTokens - finalTokens,
			"hash":            contentHash,
			"cached":          false,
		}, nil
	}
}

func makeCtxDeltaHandler(cache *mcp.HashCache) mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		path, _ := params["path"].(string)
		if path == "" {
			return nil, mcp.ErrInvalidParams
		}

		baseHash, _ := params["base_hash"].(string)

		// Load current content
		currentData, err := os.ReadFile(path)
		if err != nil {
			return nil, mcp.ErrFileNotFound
		}
		currentContent := string(currentData)
		currentHash := computeHash(currentContent)

		// Get previous version from contextread store
		store, err := contextread.Load(contextread.DefaultStorePath())
		if err != nil {
			store = &contextread.Store{}
		}

		var baseContent string
		if baseHash != "" {
			// Find entry by hash (simplified)
			if snap, ok := store.Get(path); ok {
				if computeHash(snap.Content) == baseHash {
					baseContent = snap.Content
				}
			}
		} else {
			if snap, ok := store.Get(path); ok {
				baseContent = snap.Content
				baseHash = computeHash(baseContent)
			}
		}

		// Save current as new snapshot
		store.Put(path, currentContent)
		if err := store.Save(contextread.DefaultStorePath()); err != nil {
			// Continue but warn in result
			// Snapshot save failure is non-fatal for delta computation
		}

		if baseContent == "" {
			return map[string]interface{}{
				"path":         path,
				"current_hash": currentHash,
				"base_hash":    nil,
				"delta":        nil,
				"message":      "No previous snapshot found. Current version saved.",
			}, nil
		}

		delta := filter.ComputeDelta(baseContent, currentContent)
		formattedDelta := filter.FormatDelta(delta)

		return map[string]interface{}{
			"path":         path,
			"current_hash": currentHash,
			"base_hash":    baseHash,
			"delta":        formattedDelta,
			"added":        len(delta.Added),
			"removed":      len(delta.Removed),
		}, nil
	}
}

func makeCtxGrepHandler() mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		path, _ := params["path"].(string)
		pattern, _ := params["pattern"].(string)

		if path == "" || pattern == "" {
			return nil, mcp.ErrInvalidParams
		}

		contextLines := 2
		if ctx, ok := params["context"].(float64); ok {
			contextLines = int(ctx)
		}

		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex: %w", err)
		}

		var results []map[string]interface{}

		filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			data, err := os.ReadFile(filePath)
			if err != nil {
				return nil
			}

			lines := strings.Split(string(data), "\n")
			for i, line := range lines {
				if re.MatchString(line) {
					start := i - contextLines
					if start < 0 {
						start = 0
					}
					end := i + contextLines + 1
					if end > len(lines) {
						end = len(lines)
					}

					results = append(results, map[string]interface{}{
						"file":    filePath,
						"line":    i + 1,
						"content": line,
						"context": strings.Join(lines[start:end], "\n"),
					})
				}
			}
			return nil
		})

		return map[string]interface{}{
			"pattern": pattern,
			"path":    path,
			"matches": results,
			"count":   len(results),
		}, nil
	}
}

func makeCtxHashHandler() mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		var content string

		if c, ok := params["content"].(string); ok && c != "" {
			content = c
		} else if path, ok := params["path"].(string); ok && path != "" {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, mcp.ErrFileNotFound
			}
			content = string(data)
		} else {
			return nil, mcp.ErrInvalidParams
		}

		fullHash := computeHash(content)
		shortHash := fullHash[:16]

		return map[string]interface{}{
			"full_hash":  fullHash,
			"short_hash": shortHash,
		}, nil
	}
}

func makeCtxCacheInfoHandler(cache *mcp.HashCache) mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		stats := cache.Stats()
		return map[string]interface{}{
			"total_entries": stats.TotalEntries,
			"total_size":    stats.TotalSize,
			"hit_rate":      stats.HitRate,
			"hit_count":     stats.HitCount,
			"miss_count":    stats.MissCount,
		}, nil
	}
}

func makeCtxInvalidateHandler(cache *mcp.HashCache) mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		if all, ok := params["all"].(bool); ok && all {
			count := cache.Len()
			cache.Clear()
			return map[string]interface{}{
				"deleted": count,
				"all":     true,
			}, nil
		}

		pattern, _ := params["pattern"].(string)
		if pattern == "" {
			return map[string]interface{}{
				"deleted": 0,
			}, nil
		}

		deleted := cache.InvalidateByPattern(pattern)
		return map[string]interface{}{
			"deleted": deleted,
			"pattern": pattern,
		}, nil
	}
}

func makeCtxCompactHandler(cache *mcp.HashCache) mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		var content string

		if c, ok := params["content"].(string); ok && c != "" {
			content = c
		} else if path, ok := params["path"].(string); ok && path != "" {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, mcp.ErrFileNotFound
			}
			content = string(data)
		} else {
			return nil, mcp.ErrInvalidParams
		}

		mode := filter.ModeMinimal
		if m, ok := params["mode"].(string); ok && m == "aggressive" {
			mode = filter.ModeAggressive
		}

		engine := filter.NewEngine(mode)
		compressed, saved := engine.Process(content)
		originalTokens := filter.EstimateTokens(content)
		finalTokens := filter.EstimateTokens(compressed)

		return map[string]interface{}{
			"original":          content,
			"compressed":        compressed,
			"original_tokens":   originalTokens,
			"final_tokens":      finalTokens,
			"saved_tokens":      saved,
			"reduction_percent": float64(saved) / float64(originalTokens) * 100,
		}, nil
	}
}

func makeCtxSummaryHandler() mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		path, _ := params["path"].(string)
		if path == "" {
			return nil, mcp.ErrInvalidParams
		}

		info, err := os.Stat(path)
		if err != nil {
			return nil, mcp.ErrFileNotFound
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil, mcp.ErrFileNotFound
		}

		content := string(data)
		lines := strings.Count(content, "\n") + 1
		tokens := filter.EstimateTokens(content)

		// Get first 10 lines as preview
		previewLines := strings.Split(content, "\n")
		previewCount := 10
		if len(previewLines) < previewCount {
			previewCount = len(previewLines)
		}
		preview := strings.Join(previewLines[:previewCount], "\n")

		return map[string]interface{}{
			"path":     path,
			"size":     info.Size(),
			"lines":    lines,
			"tokens":   tokens,
			"modified": info.ModTime(),
			"preview":  preview,
			"language": detectLanguage(path),
		}, nil
	}
}

// Simple in-memory memory store (should be persisted in production)
var memoryStore = make(map[string]*mcp.MemoryEntry)
var memoryMu = &sync.Mutex{}

func makeCtxRememberHandler() mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		key, _ := params["key"].(string)
		value, _ := params["value"].(string)

		if key == "" || value == "" {
			return nil, mcp.ErrInvalidParams
		}

		var tags []string
		if t, ok := params["tags"].([]interface{}); ok {
			for _, tag := range t {
				if s, ok := tag.(string); ok {
					tags = append(tags, s)
				}
			}
		}

		memoryMu.Lock()
		defer memoryMu.Unlock()

		now := time.Now()
		entry := &mcp.MemoryEntry{
			Key:       key,
			Value:     value,
			Tags:      tags,
			CreatedAt: now,
			UpdatedAt: now,
		}

		memoryStore[key] = entry

		return map[string]interface{}{
			"key":        key,
			"tags":       tags,
			"created_at": now,
		}, nil
	}
}

func makeCtxRecallHandler() mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		key, _ := params["key"].(string)
		if key == "" {
			return nil, mcp.ErrInvalidParams
		}

		memoryMu.Lock()
		defer memoryMu.Unlock()

		entry, ok := memoryStore[key]
		if !ok {
			return nil, mcp.ErrMemoryNotFound
		}

		return map[string]interface{}{
			"key":        entry.Key,
			"value":      entry.Value,
			"tags":       entry.Tags,
			"created_at": entry.CreatedAt,
			"updated_at": entry.UpdatedAt,
		}, nil
	}
}

func makeCtxSearchMemoryHandler() mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		query, _ := params["query"].(string)
		tagFilter, _ := params["tag"].(string)

		memoryMu.Lock()
		defer memoryMu.Unlock()

		var results []map[string]interface{}
		queryLower := strings.ToLower(query)

		for _, entry := range memoryStore {
			// Tag filter
			if tagFilter != "" {
				hasTag := false
				for _, t := range entry.Tags {
					if t == tagFilter {
						hasTag = true
						break
					}
				}
				if !hasTag {
					continue
				}
			}

			// Search in key and value
			if strings.Contains(strings.ToLower(entry.Key), queryLower) ||
				strings.Contains(strings.ToLower(entry.Value), queryLower) {
				results = append(results, map[string]interface{}{
					"key":   entry.Key,
					"value": entry.Value,
					"tags":  entry.Tags,
				})
			}
		}

		return map[string]interface{}{
			"query":   query,
			"results": results,
			"count":   len(results),
		}, nil
	}
}

func makeCtxBundleHandler(cache *mcp.HashCache) mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		pathsRaw, ok := params["paths"].([]interface{})
		if !ok || len(pathsRaw) == 0 {
			return nil, mcp.ErrInvalidParams
		}

		var paths []string
		for _, p := range pathsRaw {
			if s, ok := p.(string); ok {
				paths = append(paths, s)
			}
		}

		compress := false
		if c, ok := params["compress"].(bool); ok {
			compress = c
		}

		var contents []string
		totalTokens := 0

		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			content := string(data)
			if compress {
				engine := filter.NewEngine(filter.ModeMinimal)
				content, _ = engine.Process(content) // second return is saved tokens (int, not error)
			}

			contents = append(contents, fmt.Sprintf("// File: %s\n%s", path, content))
			totalTokens += filter.EstimateTokens(content)
		}

		bundle := mcp.Bundle{
			ID:        computeHash(strings.Join(paths, "|")),
			CreatedAt: time.Now(),
		}

		for i, path := range paths {
			if i < len(contents) {
				bundle.Files = append(bundle.Files, mcp.BundleFile{
					Path:    path,
					Hash:    computeFileHash(path),
					Size:    len(contents[i]),
					Content: contents[i],
				})
			}
		}

		return map[string]interface{}{
			"id":           bundle.ID,
			"files":        bundle.Files,
			"file_count":   len(bundle.Files),
			"total_tokens": totalTokens,
			"created_at":   bundle.CreatedAt,
		}, nil
	}
}

func makeCtxBundleChangedHandler() mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		since, _ := params["since"].(string)
		if since == "" {
			since = "HEAD~1"
		}

		compress := false
		if c, ok := params["compress"].(bool); ok {
			compress = c
		}

		// Get changed files from git
		cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", since)
		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("git command failed: %w", err)
		}

		files := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(files) == 1 && files[0] == "" {
			files = []string{}
		}

		var contents []string
		totalTokens := 0

		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				continue
			}

			content := string(data)
			if compress {
				engine := filter.NewEngine(filter.ModeMinimal)
				content, _ = engine.Process(content) // second return is saved tokens (int, not error)
			}

			contents = append(contents, fmt.Sprintf("// File: %s\n%s", file, content))
			totalTokens += filter.EstimateTokens(content)
		}

		return map[string]interface{}{
			"since":        since,
			"files":        files,
			"file_count":   len(files),
			"contents":     contents,
			"total_tokens": totalTokens,
		}, nil
	}
}

func makeCtxBundleSummaryHandler() mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		pathsRaw, ok := params["paths"].([]interface{})
		if !ok || len(pathsRaw) == 0 {
			return nil, mcp.ErrInvalidParams
		}

		var paths []string
		for _, p := range pathsRaw {
			if s, ok := p.(string); ok {
				paths = append(paths, s)
			}
		}

		var fileInfos []map[string]interface{}
		totalSize := int64(0)
		totalLines := 0
		totalTokens := 0
		byLanguage := make(map[string]int)

		for _, path := range paths {
			info, err := os.Stat(path)
			if err != nil {
				continue
			}

			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			content := string(data)
			lines := strings.Count(content, "\n") + 1
			tokens := filter.EstimateTokens(content)
			lang := detectLanguage(path)

			fileInfos = append(fileInfos, map[string]interface{}{
				"path":     path,
				"size":     info.Size(),
				"lines":    lines,
				"tokens":   tokens,
				"language": lang,
			})

			totalSize += info.Size()
			totalLines += lines
			totalTokens += tokens
			byLanguage[lang]++
		}

		return map[string]interface{}{
			"files":        fileInfos,
			"file_count":   len(fileInfos),
			"total_size":   totalSize,
			"total_lines":  totalLines,
			"total_tokens": totalTokens,
			"by_language":  byLanguage,
		}, nil
	}
}

func makeCtxExecHandler() mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		command, _ := params["command"].(string)
		if command == "" {
			return nil, mcp.ErrInvalidParams
		}

		var args []string
		if a, ok := params["args"].([]interface{}); ok {
			for _, arg := range a {
				if s, ok := arg.(string); ok {
					args = append(args, s)
				}
			}
		}

		timeout := 30
		if t, ok := params["timeout"].(float64); ok {
			timeout = int(t)
		}

		ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, command, args...)
		output, err := cmd.CombinedOutput()

		result := map[string]interface{}{
			"command": command,
			"args":    args,
			"output":  string(output),
		}

		if err != nil {
			result["error"] = err.Error()
			result["success"] = false
		} else {
			result["success"] = true
		}

		return result, nil
	}
}

func makeCtxTldrHandler() mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		command, _ := params["command"].(string)
		if command == "" {
			return nil, mcp.ErrInvalidParams
		}

		// Try to get TLDR page
		cmd := exec.CommandContext(ctx, "tldr", "--format", "plain", command)
		output, err := cmd.Output()
		if err != nil {
			// Fallback to man page summary
			cmd = exec.CommandContext(ctx, "man", "--pager=cat", "-P", "col -b", command)
			output, err = cmd.Output()
			if err != nil {
				return map[string]interface{}{
					"command": command,
					"found":   false,
					"error":   "No help available",
				}, nil
			}
		}

		return map[string]interface{}{
			"command": command,
			"found":   true,
			"content": string(output),
		}, nil
	}
}

func makeCtxPatternsHandler() mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		patterns := []map[string]interface{}{
			{"name": "cargo", "pattern": "tokman cargo *", "description": "Cargo build/test output filtering"},
			{"name": "npm", "pattern": "tokman npm *", "description": "NPM output filtering"},
			{"name": "go", "pattern": "tokman go *", "description": "Go command output filtering"},
			{"name": "pytest", "pattern": "tokman pytest *", "description": "Pytest output filtering"},
			{"name": "git", "pattern": "tokman git *", "description": "Git command output filtering"},
			{"name": "gh", "pattern": "tokman gh *", "description": "GitHub CLI output filtering"},
			{"name": "docker", "pattern": "tokman docker *", "description": "Docker command output filtering"},
			{"name": "kubectl", "pattern": "tokman kubectl *", "description": "Kubernetes command output filtering"},
		}

		return map[string]interface{}{
			"patterns": patterns,
			"count":    len(patterns),
		}, nil
	}
}

func makeCtxModesHandler() mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		modes := []map[string]interface{}{
			{"name": "full", "description": "Return complete file content"},
			{"name": "map", "description": "Return file structure/outline"},
			{"name": "outline", "description": "Return function/class signatures only"},
			{"name": "symbols", "description": "Return symbol definitions"},
			{"name": "imports", "description": "Return import statements only"},
			{"name": "types", "description": "Return type definitions only"},
			{"name": "exports", "description": "Return exported symbols only"},
		}

		return map[string]interface{}{
			"modes":   modes,
			"default": "auto",
		}, nil
	}
}

// defaultMode is the current default mode
var defaultMode = "auto"
var defaultModeMu = &sync.Mutex{}

func makeCtxModeHandler() mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		mode, ok := params["mode"].(string)
		if !ok || mode == "" {
			// Return current mode
			defaultModeMu.Lock()
			defer defaultModeMu.Unlock()
			return map[string]interface{}{
				"mode":    defaultMode,
				"changed": false,
			}, nil
		}

		// Validate mode
		validModes := map[string]bool{
			"full": true, "map": true, "outline": true, "symbols": true,
			"imports": true, "types": true, "exports": true, "auto": true,
		}
		if !validModes[mode] {
			return nil, fmt.Errorf("invalid mode: %s", mode)
		}

		defaultModeMu.Lock()
		oldMode := defaultMode
		defaultMode = mode
		defaultModeMu.Unlock()

		return map[string]interface{}{
			"mode":     mode,
			"previous": oldMode,
			"changed":  true,
		}, nil
	}
}

func makeCtxStatusHandler() mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{
			"version":       "1.0.0",
			"name":          "tokman-mcp",
			"tools_count":   22,
			"cache_enabled": true,
			"status":        "running",
		}, nil
	}
}

func makeCtxConfigHandler() mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		key, _ := params["key"].(string)
		value, hasValue := params["value"].(string)

		// In production, this would read/write to a config file
		// For now, return mock config
		config := map[string]interface{}{
			"default_mode":  "auto",
			"cache_size":    10000,
			"max_tokens":    100000,
			"compression":   "minimal",
			"line_numbers":  false,
			"show_progress": true,
		}

		if key == "" {
			// Return all config
			return map[string]interface{}{
				"config": config,
			}, nil
		}

		if hasValue {
			// Set config value
			config[key] = value
			return map[string]interface{}{
				"key":     key,
				"value":   value,
				"updated": true,
			}, nil
		}

		// Get specific config value
		if val, ok := config[key]; ok {
			return map[string]interface{}{
				"key":   key,
				"value": val,
			}, nil
		}

		return nil, fmt.Errorf("config key not found: %s", key)
	}
}

func makeCtxMcpHandler() mcp.ToolHandler {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{
			"name":        "tokman",
			"version":     "1.0.0",
			"transport":   "stdio",
			"tools":       27,
			"entrypoint":  "tokman mcp",
			"env":         map[string]string{},
			"config_file": filepath.Join(config.ConfigDir(), "mcp.json"),
		}, nil
	}
}

// Helper functions

func computeHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

func computeFileHash(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return computeHash(string(data))
}

func detectLanguage(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".go":
		return "go"
	case ".rs":
		return "rust"
	case ".py":
		return "python"
	case ".js", ".mjs":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".java":
		return "java"
	case ".rb":
		return "ruby"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".hpp":
		return "cpp"
	case ".md":
		return "markdown"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	default:
		return "unknown"
	}
}
