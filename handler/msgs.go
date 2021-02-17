package handler

import (
	"collab-chess/game"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strings"

	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"
)

// Msg interface includes methods about important info for a slack message and handles what to do with it
type Msg interface {
	ChannelID() string
	Timestamp() string
	ThreadTimestamp() string
	Raw() *slackevents.MessageEvent

	Handle(s *SlackHandler)
}

// GameStartMsg is a struct for a message to start a new game
type GameStartMsg struct {
	player string
	raw    *slackevents.MessageEvent
}

func (m GameStartMsg) ChannelID() string {
	return m.raw.Channel
}

func (m GameStartMsg) Timestamp() string {
	return m.raw.TimeStamp
}

func (m GameStartMsg) ThreadTimestamp() string {
	return m.raw.ThreadTimeStamp
}

func (m GameStartMsg) Raw() *slackevents.MessageEvent {
	return m.raw
}

func ParseGameStartMsg(m *slackevents.MessageEvent) (*GameStartMsg, bool) {
	// cannot be in a thread
	if m.ThreadTimeStamp != "" {
		return nil, false
	}

	// it is in a DM
	if strings.HasPrefix(m.Channel, "D") {
		return nil, false
	}

	if m.Text == "!start" {
		return &GameStartMsg{raw: m, player: m.User}, true
	}

	return nil, false
}

// generates a random integer between min and max
func randomInt(min, max int) int {
	return min + rand.Intn(max-min)
}

// Generate a random string of A-Z chars with len = l
func randomString(len int) string {
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		bytes[i] = byte(randomInt(97, 122))
	}
	return string(bytes)
}

func (msg GameStartMsg) Handle(s *SlackHandler) {
	log.Println(msg.player, "is starting a chess game")

	_, err := s.GameStorage.RetrieveGame()
	if err == nil {
		s.SlackClient.PostMessage(msg.ChannelID(), slack.MsgOptionText("There is already a game in place. Make your move!", false))
		return
	}

	players := []game.Player{
		{ID: "chessbot"},
		{ID: msg.player},
	}

	gameID := randomString(20)

	gm := game.NewGame(gameID, players...)
	s.GameStorage.StoreGame(gm)

	go s.GameLoop()

	humanColor, err := gm.GetColor(msg.player)
	text := fmt.Sprintf("Hackalackers are playing: %s", humanColor)
	s.SlackClient.PostMessage(msg.ChannelID(), slack.MsgOptionText(text, false))
}

// MoveMsg represents a move
type MoveMsg struct {
	san    string
	player string
	raw    *slackevents.MessageEvent
}

func (m MoveMsg) ChannelID() string {
	return m.raw.Channel
}

func (m MoveMsg) Timestamp() string {
	return m.raw.TimeStamp
}

func (m MoveMsg) ThreadTimestamp() string {
	return m.raw.ThreadTimeStamp
}

func (m MoveMsg) Raw() *slackevents.MessageEvent {
	return m.raw
}

func ParseMoveMsg(m *slackevents.MessageEvent) (*MoveMsg, bool) {
	// cannot be in a thread
	if m.ThreadTimeStamp != "" {
		return nil, false
	}

	// it is in a DM
	if strings.HasPrefix(m.Channel, "D") {
		return nil, false
	}

	regex := regexp.MustCompile("^!move (.*)$")
	matches := regex.FindStringSubmatch(m.Text)
	if matches == nil {
		return nil, false
	}

	playerMove := matches[1]

	return &MoveMsg{san: playerMove, player: m.User, raw: m}, true
}

func (msg MoveMsg) Handle(s *SlackHandler) {
	gm, err := s.GameStorage.RetrieveGame()

	if err != nil {
		s.SlackClient.PostMessage(msg.ChannelID(), slack.MsgOptionText("There isn't an active game at the moment :( You can use the *!start* command to start a new game :chess_pawn: ", false))
		return
	}

	// if our mutex locks are properly working this should be redundant
	if gm.TurnPlayer().ID == "chessbot" {
		return
	}

	moveErr := gm.Vote(msg.player, msg.san)

	if moveErr != nil {
		fmt.Println(moveErr)
		return
	}
}

// BoardMsg represents a message to ask the current board state
type BoardMsg struct {
	player string
	raw    *slackevents.MessageEvent
}

func (m BoardMsg) ChannelID() string {
	return m.raw.Channel
}

func (m BoardMsg) Timestamp() string {
	return m.raw.TimeStamp
}

func (m BoardMsg) ThreadTimestamp() string {
	return m.raw.ThreadTimeStamp
}

func (m BoardMsg) Raw() *slackevents.MessageEvent {
	return m.raw
}

func ParseBoardMsg(m *slackevents.MessageEvent) (*BoardMsg, bool) {
	// cannot be in a thread
	if m.ThreadTimeStamp != "" {
		return nil, false
	}

	// it is in a DM
	if strings.HasPrefix(m.Channel, "D") {
		return nil, false
	}

	if m.Text == "!board" {
		return &BoardMsg{raw: m, player: m.User}, true
	}

	return nil, false
}

func (m BoardMsg) Handle(s *SlackHandler) {
	gm, err := s.GameStorage.RetrieveGame()

	if err != nil {
		return
	}

	gm.Lock()
	defer gm.Unlock()
	link, _ := s.LinkRenderer.CreateLink(gm)

	boardAttachment := slack.Attachment{
		ImageURL: link.String(),
		Color:    colorToHex[gm.Turn()],
	}

	s.SlackClient.PostMessage(s.GameChannel, slack.MsgOptionText("Here is the current state of the game", false), slack.MsgOptionAttachments(boardAttachment))
}

// This parses messages to either a msg to start the game or to play a move
func parseMessage(msg *slackevents.MessageEvent) Msg {
	var parsed Msg
	var ok bool

	parsed, ok = ParseGameStartMsg(msg)
	if ok {
		return parsed
	}

	parsed, ok = ParseMoveMsg(msg)
	if ok {
		return parsed
	}

	parsed, ok = ParseBoardMsg(msg)
	if ok {
		return parsed
	}

	return nil

}
