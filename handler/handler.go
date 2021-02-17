package handler

import (
	"collab-chess/game"
	"collab-chess/rendering"
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"
	"github.com/notnil/chess"
	"github.com/notnil/chess/uci"
)

// SlackHandler handles Slack events
type SlackHandler struct {
	SigningKey   string
	BotToken     string
	SlackClient  *slack.Client
	GameStorage  game.ChessStorage
	LinkRenderer rendering.RenderLink
	GameChannel  string
}

var colorToHex = map[game.Color]string{
	game.Black: "#000000",
	game.White: "#eeeeee",
}

func (s SlackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sv, err := slack.NewSecretsVerifier(r.Header, s.SigningKey)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, err := sv.Write(body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := sv.Ensure(); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if eventsAPIEvent.Type == slackevents.URLVerification {
		var r *slackevents.ChallengeResponse
		err := json.Unmarshal([]byte(body), &r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text")
		w.Write([]byte(r.Challenge))
	}
	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		if s.SlackClient == nil {
			s.SlackClient = slack.New(s.BotToken)
		}

		innerEvent := eventsAPIEvent.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			s.SlackClient.PostMessage(ev.Channel, slack.MsgOptionText("Hi! I live in #playchess at Hack Club. You can type !start to start a game of chess, !move [notation] (for example, !move e4 or !move Nc6) to vote on a move. Each turn top voted move gets played. If no votes are present after 3 mins, current game stops. Good luck! :chess_pawn:", false))
		case *slackevents.MessageEvent:
			msg := parseMessage(ev)
			if msg == nil {
				return
			}

			msg.Handle(&s)
		}
	}
}

// GameLoop is the main loop where the game starts and checks for moves between players
func (s SlackHandler) GameLoop() {
	// set up engine to use stockfish exe
	eng, err := uci.New("stockfish")
	if err != nil {
		panic(err)
	}

	defer eng.Close()
	// initialize uci with new game
	if err := eng.Run(uci.CmdUCI, uci.CmdIsReady, uci.CmdUCINewGame); err != nil {
		panic(err)
	}

	func() {
		for {
			time.Sleep(time.Second)

			gm, err := s.GameStorage.RetrieveGame()

			if err != nil {
				return
			}

			if outcome := gm.Outcome(); outcome != chess.NoOutcome {

				link, _ := s.LinkRenderer.CreateLink(gm)

				boardAttachment := slack.Attachment{
					ImageURL: link.String(),
					Color:    colorToHex[gm.Turn()],
				}

				s.SlackClient.PostMessage(s.GameChannel, slack.MsgOptionText(gm.ResultText(), false), slack.MsgOptionAttachments(boardAttachment))
				s.GameStorage.RemoveGame()
				return
			}

			if gm.TurnPlayer().ID == "chessbot" {
				gm.Lock()
				cmdPos := uci.CmdPosition{Position: gm.Position()}

				// thinkingTime is a value between 10 and 60
				// to simulate the thinking time of our bot so that we get different moves
				thinkingTime := time.Duration(rand.Intn(51) + 10)

				cmdGo := uci.CmdGo{MoveTime: 2 * time.Second / thinkingTime}
				if err := eng.Run(cmdPos, cmdGo); err != nil {
					panic(err)
				}
				move := eng.SearchResults().BestMove
				gm.Unlock()
				if err := gm.BotMove(move); err != nil {
					panic(err)
				}

				if outcome := gm.Outcome(); outcome != chess.NoOutcome {
					continue
				}

				link, _ := s.LinkRenderer.CreateLink(gm)

				boardAttachment := slack.Attachment{
					ImageURL: link.String(),
					Color:    colorToHex[gm.Turn()],
				}

				s.SlackClient.PostMessage(s.GameChannel, slack.MsgOptionText("I made my move :crossed_swords:", false), slack.MsgOptionAttachments(boardAttachment))
			}

			if gm.TurnPlayer().ID != "chessbot" {
				if time.Since(gm.LastMoveTime()) > 3*time.Minute {
					log.Println("nobody made a move :( removing the current game from pool")
					s.GameStorage.RemoveGame()
					return
				}

				if time.Since(gm.LastMoveTime()) > 1*time.Minute {
					_, err := gm.MoveTopVote()
					if err != nil {
						continue
					}
				}
			}
		}
	}()
}
