package game

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/notnil/chess"
)

// uniqueVoters is a slice that holds players who voted during a game
type uniqueVoters []string

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
	playersVoted uniqueVoters
	lastMoved    time.Time
	firstVoted   time.Time
	checkedTile  *chess.Square
	timeProvider TimeProvider
	sync.Mutex
}

// Player represents a human Chess player
type Player struct {
	ID    string
	color Color
}

// NewGame creates and returns a new game
func NewGame(ID string, pieceColor string, players ...Player) *Game {
	gm := &Game{
		ID:           ID,
		game:         chess.NewGame(),
		lastMoved:    time.Now(),
		firstVoted:   time.Now(),
		votes:        make(map[string]string),
		playersVoted: uniqueVoters{},
		timeProvider: defaultTimeProvider,
	}

	attachPlayers(gm, pieceColor, players...)

	return gm
}

func attachPlayers(g *Game, pieceColor string, players ...Player) {
	playerList := []Player{}
	playerList = append(playerList, players...)

	if pieceColor == "" {
		rand.Shuffle(2, func(i, j int) {
			playerList[i], playerList[j] = playerList[j], playerList[i]
		})
		playerList[0].color = White
		playerList[1].color = Black
		g.Players = map[Color]Player{
			White: playerList[0],
			Black: playerList[1],
		}
	} else if pieceColor == "white" {
		playerList[0].color = Black
		playerList[1].color = White

		g.Players = map[Color]Player{
			White: playerList[1],
			Black: playerList[0],
		}
	} else if pieceColor == "black" {
		playerList[0].color = White
		playerList[1].color = Black

		g.Players = map[Color]Player{
			White: playerList[0],
			Black: playerList[1],
		}
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
	g.Lock()
	defer g.Unlock()
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

	if winningPlayer.ID != "chessbot" {
		uniquePlayers := g.playersVoted
		return fmt.Sprintf("%s %s by %s", uniquePlayers, g.Outcome(), g.game.Method())
	}

	return fmt.Sprintf("I won this time :chess_pawn: Better luck next time! %s by %s", g.Outcome(), g.game.Method())

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

// FirstVoteTime returns the time of the first vote
func (g *Game) FirstVoteTime() time.Time {
	return g.firstVoted
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
	g.Lock()
	defer g.Unlock()
	// this returns an error if it is not a valid move
	_, err := chess.AlgebraicNotation{}.Decode(g.game.Position(), move)

	if err != nil {
		return fmt.Errorf("move is not valid")
	}

	_, ok := g.votes[playerID]
	if !ok {
		log.Println(playerID, "is making a move:", move)
		g.votes[playerID] = move
	}

	// if this was the first vote then we update the firstVoted
	if len(g.votes) == 1 {
		g.firstVoted = g.timeProvider()
	}

	username := fmt.Sprintf("<@%s>", playerID)
	for _, val := range g.playersVoted {
		if username == val {
			return nil
		}
	}

	g.playersVoted = append(g.playersVoted, username)
	return nil
}

// MoveTopVote moves the top voted piece
func (g *Game) MoveTopVote() (string, error) {
	g.Lock()
	defer g.Unlock()

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
		return "", fmt.Errorf("there was no top vote")
	}

	_, err := g.Move(topVote)

	if err != nil {
		return "", fmt.Errorf("there was a problem playing the move %s", topVote)
	}

	// reset votes after the voting
	g.votes = map[string]string{}
	return topVote, nil
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

// Stringer for uniqueVoters
func (uv uniqueVoters) String() string {
	players := strings.Join(uv, ", ")
	switch len(uv) {
	case 1:
		return fmt.Sprintf("Well played! %s. You defeated the bot!", players)
	default:
		return fmt.Sprintf("Good job everyone! %s defeated the bot together!", players)
	}
}
