package toml

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// CommunityFilterRegistry manages community filter sharing and publishing.
type CommunityFilterRegistry struct {
	mu        sync.RWMutex
	localPath string
	remoteURL string
	filters   []RegistryEntry
}

// RegistryEntry represents a published filter in the registry.
type RegistryEntry struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Author      string    `json:"author"`
	Version     string    `json:"version"`
	Category    string    `json:"category"`
	Downloads   int       `json:"downloads"`
	Rating      float64   `json:"rating"`
	Tags        []string  `json:"tags,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Content     string    `json:"content"`
	SHA256      string    `json:"sha256"`
}

// NewCommunityFilterRegistry creates a new community filter registry.
func NewCommunityFilterRegistry(localPath, remoteURL string) *CommunityFilterRegistry {
	r := &CommunityFilterRegistry{
		localPath: localPath,
		remoteURL: remoteURL,
		filters:   []RegistryEntry{},
	}
	r.loadLocal()
	return r
}

// Publish publishes a filter to the registry.
func (r *CommunityFilterRegistry) Publish(entry RegistryEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry.CreatedAt = time.Now()
	entry.UpdatedAt = time.Now()
	safety := CheckFilterSafety(entry.Content)
	if !safety.Passed {
		return fmt.Errorf("filter failed safety checks: %v", safety.Issues)
	}
	r.filters = append(r.filters, entry)
	return r.saveLocal()
}

// List returns all available filters, optionally filtered by category.
func (r *CommunityFilterRegistry) List(category string) []RegistryEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if category == "" {
		return r.filters
	}
	var filtered []RegistryEntry
	for _, f := range r.filters {
		if strings.EqualFold(f.Category, category) {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

// Search searches filters by name, description, or tags.
func (r *CommunityFilterRegistry) Search(query string) []RegistryEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	query = strings.ToLower(query)
	var results []RegistryEntry
	for _, f := range r.filters {
		if strings.Contains(strings.ToLower(f.Name), query) ||
			strings.Contains(strings.ToLower(f.Description), query) {
			results = append(results, f)
			continue
		}
		for _, tag := range f.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				results = append(results, f)
				break
			}
		}
	}
	return results
}

// Install downloads and installs a filter by name.
func (r *CommunityFilterRegistry) Install(name string) error {
	r.mu.RLock()
	var entry *RegistryEntry
	for i := range r.filters {
		if r.filters[i].Name == name {
			entry = &r.filters[i]
			break
		}
	}
	r.mu.RUnlock()
	if entry == nil {
		return fmt.Errorf("filter %q not found", name)
	}
	dir := filepath.Dir(r.localPath)
	os.MkdirAll(dir, 0755)
	destPath := filepath.Join(dir, name+".toml")
	return os.WriteFile(destPath, []byte(entry.Content), 0644)
}

// Eject copies a built-in filter to local config for customization.
func (r *CommunityFilterRegistry) Eject(name string, destDir string) error {
	r.mu.RLock()
	var content string
	for _, f := range r.filters {
		if f.Name == name {
			content = f.Content
			break
		}
	}
	r.mu.RUnlock()
	if content == "" {
		return fmt.Errorf("filter %q not found", name)
	}
	os.MkdirAll(destDir, 0755)
	return os.WriteFile(filepath.Join(destDir, name+".toml"), []byte(content), 0644)
}

// Stats returns registry statistics.
func (r *CommunityFilterRegistry) Stats() map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	byCategory := make(map[string]int)
	for _, f := range r.filters {
		byCategory[f.Category]++
	}
	return map[string]any{
		"total_filters": len(r.filters),
		"by_category":   byCategory,
	}
}

func (r *CommunityFilterRegistry) loadLocal() {
	if r.localPath == "" {
		return
	}
	data, err := os.ReadFile(r.localPath)
	if err != nil {
		return
	}
	json.Unmarshal(data, &r.filters)
}

func (r *CommunityFilterRegistry) saveLocal() error {
	if r.localPath == "" {
		return nil
	}
	dir := filepath.Dir(r.localPath)
	os.MkdirAll(dir, 0755)
	data, _ := json.MarshalIndent(r.filters, "", "  ")
	return os.WriteFile(r.localPath, data, 0644)
}

// FormatRegistryStats returns a human-readable stats string.
func FormatRegistryStats(stats map[string]any) string {
	return fmt.Sprintf("Registry: %d filters, %d categories",
		stats["total_filters"], len(stats["by_category"].(map[string]int)))
}
