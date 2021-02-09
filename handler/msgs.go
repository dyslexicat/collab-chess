package handler

import (
	"chess-slack/game"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"
)

// Msg parses a slack message and handles what to do with it
type Msg interface {
	ChannelID() string
	Timestamp() string
	ThreadTimestamp() string
	Raw() *slackevents.MessageEvent

	Handle(s *SlackHandler)
}

// GameStartMsg starts the game
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

func (msg GameStartMsg) Handle(s *SlackHandler) {
	log.Println(msg.player, "is starting a chess game")

	_, err := s.GameStorage.RetrieveGame()
	if err == nil {
		s.SlackClient.PostMessage(msg.ChannelID(), slack.MsgOptionText("There is already a game in place", false))
		return
	}

	players := []game.Player{
		{ID: "bot"},
		{ID: msg.player},
	}

	gm := game.NewGame("1234", players...)
	s.GameStorage.StoreGame(gm)

	go s.GameLoop()

	humanColor, err := gm.GetColor(msg.player)
	text := fmt.Sprintf("Human is %s", humanColor)
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
		return
	}

	if gm.TurnPlayer().ID == "bot" {
		log.Println("it is the bot's turn")
		return
	}

	moveErr := gm.Vote(msg.player, msg.san)

	if moveErr != nil {
		fmt.Println(moveErr)
		return
	}
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

	return nil

}
