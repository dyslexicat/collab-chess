package handler

import (
	"chess-slack/game"
	"chess-slack/rendering"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
			fmt.Println("mention event: ", ev)
			s.SlackClient.PostMessage(ev.Channel, slack.MsgOptionText("hello", false))
		case *slackevents.MessageEvent:
			if ev.Text == "!exit" {
				s.GameStorage.RemoveGame()
				return
			}

			msg := parseMessage(ev)
			if msg == nil {
				return
			}

			msg.Handle(&s)

			//fmt.Println("message event", ev)
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

	for {
		gm, err := s.GameStorage.RetrieveGame()

		if err != nil {
			return
		}

		if outcome := gm.Outcome(); outcome != chess.NoOutcome {
			s.SlackClient.PostMessage("C01GNJRCQLD", slack.MsgOptionText(gm.ResultText(), false))
			s.GameStorage.RemoveGame()
			return
		}

		if gm.TurnPlayer().ID == "bot" {
			time.Sleep(time.Second * 2)
			cmdPos := uci.CmdPosition{Position: gm.Position()}
			cmdGo := uci.CmdGo{MoveTime: time.Second / 100}
			if err := eng.Run(cmdPos, cmdGo); err != nil {
				panic(err)
			}
			move := eng.SearchResults().BestMove
			if err := gm.BotMove(move); err != nil {
				panic(err)
			}

			link, _ := s.LinkRenderer.CreateLink(gm)

			boardAttachment := slack.Attachment{
				ImageURL: link.String(),
				Color:    colorToHex[gm.Turn()],
			}

			s.SlackClient.PostMessage("C01GNJRCQLD", slack.MsgOptionText("bot played", false), slack.MsgOptionAttachments(boardAttachment))
			fmt.Println(gm)
		}

		if gm.TurnPlayer().ID != "bot" {
			if time.Since(gm.LastMoveTime()) > 30*time.Second {
				fmt.Println("removing the current game from pool")
				s.GameStorage.RemoveGame()
				g, _ := s.GameStorage.RetrieveGame()
				fmt.Println(g)
			}

			if time.Since(gm.LastMoveTime()) > 20*time.Second {
				_, err := gm.MoveTopVote()
				if err != nil {
					continue
				}

				//api.PostMessage("C01GNJRCQLD", slack.MsgOptionText(, false))
				fmt.Println(gm)
			}
		}
	}
}
