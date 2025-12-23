package game

import (
	"fmt"
	"sync"
)

// Registry manages game registration and lookup.
// It provides a thread-safe way to register and retrieve games by their command.
// Requirements: 10.2 - Plugin-style game registration
type Registry struct {
	games map[string]Game
	mu    sync.RWMutex
}

// NewRegistry creates a new game registry.
func NewRegistry() *Registry {
	return &Registry{
		games: make(map[string]Game),
	}
}

// Register adds a game to the registry.
// If a game with the same command already exists, it will be replaced.
// Requirements: 10.2
func (r *Registry) Register(g Game) error {
	if g == nil {
		return fmt.Errorf("cannot register nil game")
	}
	if g.Command() == "" {
		return fmt.Errorf("game command cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.games[g.Command()] = g
	return nil
}

// Get retrieves a game by its command.
// Returns the game and true if found, nil and false otherwise.
// Requirements: 10.2
func (r *Registry) Get(command string) (Game, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	g, ok := r.games[command]
	return g, ok
}

// List returns all registered games.
// The returned slice is a copy, so modifications won't affect the registry.
// Requirements: 10.2
func (r *Registry) List() []Game {
	r.mu.RLock()
	defer r.mu.RUnlock()

	games := make([]Game, 0, len(r.games))
	for _, g := range r.games {
		games = append(games, g)
	}
	return games
}

// Commands returns all registered game commands.
func (r *Registry) Commands() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	commands := make([]string, 0, len(r.games))
	for cmd := range r.games {
		commands = append(commands, cmd)
	}
	return commands
}

// Count returns the number of registered games.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.games)
}

// Unregister removes a game from the registry by its command.
// Returns true if the game was found and removed, false otherwise.
func (r *Registry) Unregister(command string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.games[command]; ok {
		delete(r.games, command)
		return true
	}
	return false
}

// DefaultRegistry is the global game registry instance.
var DefaultRegistry = NewRegistry()

// Register adds a game to the default registry.
func Register(g Game) error {
	return DefaultRegistry.Register(g)
}

// GetGame retrieves a game from the default registry.
func GetGame(command string) (Game, bool) {
	return DefaultRegistry.Get(command)
}

// ListGames returns all games from the default registry.
func ListGames() []Game {
	return DefaultRegistry.List()
}
