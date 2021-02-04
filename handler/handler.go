package handler

import (
	"chess-slack/game"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"
)

// SlackHandler handles Slack events
type SlackHandler struct {
	SigningKey  string
	BotToken    string
	SlackClient *slack.Client
	GameStorage game.ChessStorage
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
			regex := regexp.MustCompile("^!move (.*)$")
			matches := regex.FindStringSubmatch(ev.Text)

			if ev.Text == "!start" {
				_, err := s.GameStorage.RetrieveGame()
				if err == nil {
					s.SlackClient.PostMessage(ev.Channel, slack.MsgOptionText("there is already a game in place", false))
					return
				}

				players := []game.Player{
					{ID: "bot"},
					{ID: "alp"},
				}

				gm := game.NewGame("1234", players...)
				s.GameStorage.StoreGame(gm)

				humanColor, err := gm.GetColor("alp")
				msg := fmt.Sprintf("Human is %s", humanColor)
				s.SlackClient.PostMessage(ev.Channel, slack.MsgOptionText(msg, false))
			}

			if ev.Text == "!exit" {
				s.GameStorage.RemoveGame()
				return
			}

			gm, err := s.GameStorage.RetrieveGame()

			if err != nil {
				fmt.Println(err)
				return
			}

			if len(matches) > 0 && gm.TurnPlayer().ID == "alp" {
				playerMove := matches[1]
				fmt.Println(playerMove)

				err := gm.Vote(playerMove)

				if err != nil {
					fmt.Println(err)
					return
				}

				fmt.Println(gm.Votes())

			}

			//fmt.Println("message event", ev)
		}
	}
}
