// Package favorites provides favorite commands management
package favorites

import (
	"fmt"
	"sync"
	"time"
)

// FavoriteCommand represents a favorite command
type FavoriteCommand struct {
	ID          string
	Name        string
	Command     string
	Description string
	Tags        []string
	UsageCount  int
	LastUsed    time.Time
	CreatedAt   time.Time
}

// FavoritesManager manages favorite commands
type FavoritesManager struct {
	favorites map[string]*FavoriteCommand
	mu        sync.RWMutex
}

// NewFavoritesManager creates a new favorites manager
func NewFavoritesManager() *FavoritesManager {
	return &FavoritesManager{
		favorites: make(map[string]*FavoriteCommand),
	}
}

// AddFavorite adds a favorite command
func (fm *FavoritesManager) AddFavorite(fav *FavoriteCommand) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if fav.ID == "" {
		fav.ID = fmt.Sprintf("fav-%d", time.Now().UnixNano())
	}
	if fav.CreatedAt.IsZero() {
		fav.CreatedAt = time.Now()
	}
	if fav.Tags == nil {
		fav.Tags = make([]string, 0)
	}

	fm.favorites[fav.ID] = fav
	return nil
}

// RemoveFavorite removes a favorite command
func (fm *FavoritesManager) RemoveFavorite(id string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if _, ok := fm.favorites[id]; !ok {
		return fmt.Errorf("favorite not found: %s", id)
	}

	delete(fm.favorites, id)
	return nil
}

// GetFavorite returns a favorite by ID
func (fm *FavoritesManager) GetFavorite(id string) (*FavoriteCommand, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	fav, ok := fm.favorites[id]
	if !ok {
		return nil, fmt.Errorf("favorite not found: %s", id)
	}

	return fav, nil
}

// ListFavorites returns all favorites
func (fm *FavoritesManager) ListFavorites() []*FavoriteCommand {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	favs := make([]*FavoriteCommand, 0, len(fm.favorites))
	for _, fav := range fm.favorites {
		favs = append(favs, fav)
	}

	return favs
}

// GetByTag returns favorites with a specific tag
func (fm *FavoritesManager) GetByTag(tag string) []*FavoriteCommand {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	favs := make([]*FavoriteCommand, 0)
	for _, fav := range fm.favorites {
		for _, t := range fav.Tags {
			if t == tag {
				favs = append(favs, fav)
				break
			}
		}
	}

	return favs
}

// IncrementUsage increments usage count
func (fm *FavoritesManager) IncrementUsage(id string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if fav, ok := fm.favorites[id]; ok {
		fav.UsageCount++
		fav.LastUsed = time.Now()
	}
}

// GetMostUsed returns the most used favorites
func (fm *FavoritesManager) GetMostUsed(count int) []*FavoriteCommand {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	favs := make([]*FavoriteCommand, 0, len(fm.favorites))
	for _, fav := range fm.favorites {
		favs = append(favs, fav)
	}

	// Sort by usage count (simple bubble sort)
	for i := 0; i < len(favs); i++ {
		for j := i + 1; j < len(favs); j++ {
			if favs[j].UsageCount > favs[i].UsageCount {
				favs[i], favs[j] = favs[j], favs[i]
			}
		}
	}

	if count > len(favs) {
		count = len(favs)
	}

	return favs[:count]
}
