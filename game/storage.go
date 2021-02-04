package game

// ChessStorage is an interface to persist a game
type ChessStorage interface {
	RetrieveGame() (*Game, error)
	StoreGame(game *Game) error
	RemoveGame() error
}
