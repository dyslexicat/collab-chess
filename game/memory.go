package game

import "fmt"

// MemoryStore implements the GameStore interface and holds the state in memory
type MemoryStore struct {
	game *Game
}

// NewMemoryStore returns a MemoryStore pointer
func NewMemoryStore() *MemoryStore {
	store := MemoryStore{game: nil}
	return &store
}

// RetrieveGame returns the game from the store
func (m *MemoryStore) RetrieveGame() (*Game, error) {
	if m.game == nil {
		return nil, fmt.Errorf("There is no game at the moment")
	}

	return m.game, nil
}

// StoreGame stores the game in the store
func (m *MemoryStore) StoreGame(game *Game) error {
	m.game = game
	return nil
}

// RemoveGame deletes the active game
func (m *MemoryStore) RemoveGame() error {
	m.game = nil
	return nil
}
