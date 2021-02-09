package game

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/notnil/chess"
)

// Color represents the piece colors
type Color string

// White for white pieces and Black for black pieces
const (
	White Color = "White"
	Black Color = "Black"
)

var ColorMap = map[Color]chess.Color{
	White: chess.White,
	Black: chess.Black,
}

// TimeProvider is a closure that returns the current time as determined by the provider
type TimeProvider func() time.Time

var defaultTimeProvider TimeProvider = func() time.Time {
	return time.Now()
}

// Game is a chess game
type Game struct {
	ID           string
	game         *chess.Game
	started      bool
	Players      map[Color]Player
	votes        map[string]string
	lastMoved    time.Time
	checkedTile  *chess.Square
	timeProvider TimeProvider
	mu           sync.Mutex
}

// Player represents a human Chess player
type Player struct {
	ID    string
	color Color
}

// NewGame creates and returns a new game
func NewGame(ID string, players ...Player) *Game {
	gm := &Game{
		ID:           ID,
		game:         chess.NewGame(),
		lastMoved:    time.Now(),
		votes:        make(map[string]string),
		timeProvider: defaultTimeProvider,
	}
	attachPlayers(gm, players...)
	return gm
}

func attachPlayers(g *Game, players ...Player) {
	playerList := []Player{}
	playerList = append(playerList, players...)
	rand.Shuffle(2, func(i, j int) {
		playerList[i], playerList[j] = playerList[j], playerList[i]
	})
	playerList[0].color = White
	playerList[1].color = Black
	g.Players = map[Color]Player{
		White: playerList[0],
		Black: playerList[1],
	}
}

// TurnPlayer returns which player should move next
func (g *Game) TurnPlayer() Player {
	return g.Players[g.Turn()]
}

// Turn returns which color should move next
func (g *Game) Turn() Color {
	switch g.game.Position().Turn() {
	case chess.White:
		return White
	case chess.Black:
		return Black
	default:
		return White
	}
}

// FEN serializer
func (g *Game) FEN() string {
	return g.game.FEN()
}

// GetColor returns the piece color for a given ID
func (g *Game) GetColor(ID string) (Color, error) {
	for key, val := range g.Players {
		if val.ID == ID {
			return key, nil
		}
	}
	return "", fmt.Errorf("this player does not exist in this game")
}

// Move a Chess piece based on standard algebraic notation (d2d4, etc)
func (g *Game) Move(san string) (*chess.Move, error) {
	err := g.game.MoveStr(san)
	if err != nil {
		return nil, err
	}
	g.started = true
	g.lastMoved = g.timeProvider()
	return g.LastMove(), nil
}

// BotMove simulates a move for our bot player
func (g *Game) BotMove(m *chess.Move) error {
	err := g.game.Move(m)
	g.started = true
	g.lastMoved = g.timeProvider()
	return err
}

// Outcome determines the outcome of the game (or no outcome)
func (g *Game) Outcome() chess.Outcome {
	return g.game.Outcome()
}

// ResultText will show the outcome of the game in textual format
func (g *Game) ResultText() string {
	outcome := g.Outcome()
	if outcome == chess.Draw {
		return fmt.Sprintf("Game completed. %s by %s.", g.Outcome(), g.game.Method())
	}
	var winningPlayer Player
	if outcome == chess.WhiteWon {
		winningPlayer = g.Players[White]
	} else {
		winningPlayer = g.Players[Black]
	}
	return fmt.Sprintf("Congratulations, <@%v>! %s by %s", winningPlayer.ID, g.Outcome(), g.game.Method())
}

// LastMove returns the last move done of the game
func (g *Game) LastMove() *chess.Move {
	moves := g.game.Moves()
	if len(moves) == 0 {
		return nil
	}
	return moves[len(moves)-1]
}

// LastMoveTime returns the time when last piece was moved
func (g *Game) LastMoveTime() time.Time {
	return g.lastMoved
}

// Start indicates the game has been started
func (g *Game) Start() {
	g.started = true
}

// Started determines if the game has been started
func (g *Game) Started() bool {
	return g.started
}

// ValidMoves returns a list of all moves available to the current player's turn
func (g *Game) ValidMoves() []*chess.Move {
	return g.game.ValidMoves()
}

// Board representation as a string
func (g *Game) String() string {
	return g.game.Position().Board().Draw()
}

// Position returns the current position of the board for the engine move
func (g *Game) Position() *chess.Position {
	return g.game.Position()
}

// Votes returns the voted moves so far
func (g *Game) Votes() map[string]string {
	return g.votes
}

// Vote votes on a move if it is a valid move
func (g *Game) Vote(playerID string, move string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	// this returns an error if it is not a valid move
	_, err := chess.AlgebraicNotation{}.Decode(g.game.Position(), move)

	if err != nil {
		return fmt.Errorf("move is not valid")
	}

	_, ok := g.votes[playerID]
	if !ok {
		log.Println(playerID, " is making a move.")
		g.votes[playerID] = move
	}

	return nil
}

// MoveTopVote moves the top voted piece
func (g *Game) MoveTopVote() (*chess.Move, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	freqs := make(map[string]int)
	for _, move := range g.votes {
		freqs[move]++
	}

	var topVote string
	var topVoteCount int

	for key, val := range freqs {
		if val >= topVoteCount {
			topVote = key
			topVoteCount = val
		}
	}

	if topVoteCount == 0 {
		return nil, fmt.Errorf("there was no top vote")
	}

	chessMove, err := g.Move(topVote)

	if err != nil {
		return nil, fmt.Errorf("there was a problem playing the move %s", topVote)
	}

	// reset votes after the voting
	g.votes = map[string]string{}
	return chessMove, nil
}

// CheckedKing returns the square of a checked king if there is indeed a king in check.
func (g *Game) CheckedKing() chess.Square {
	squareMap := g.game.Position().Board().SquareMap()
	lastMovePiece := squareMap[g.LastMove().S2()]
	for square, piece := range squareMap {
		if piece.Type() == chess.King && piece.Color() == lastMovePiece.Color().Other() {
			return square
		}
	}
	return chess.NoSquare
}
