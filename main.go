package main

import (
	"chess-slack/game"
	"chess-slack/handler"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/nlopes/slack"
	"github.com/notnil/chess"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("error loading .env files")
	}

	// slack api bot token (xobx...)
	slackAuthToken := os.Getenv("SLACK_BOT_TOKEN")
	signingSecret := os.Getenv("SLACK_SIGNING_SECRET")

	var gameStorage game.ChessStorage
	api := slack.New(slackAuthToken)

	memoryStore := game.NewMemoryStore()
	gameStorage = memoryStore

	sHandler := handler.SlackHandler{
		SigningKey:  signingSecret,
		BotToken:    slackAuthToken,
		GameStorage: gameStorage,
	}

	http.Handle("/slack/events", sHandler)

	go func() {
		for {
			gm, err := sHandler.GameStorage.RetrieveGame()

			if err != nil {
				continue
			}

			if outcome := gm.Outcome(); outcome != chess.NoOutcome {
				api.PostMessage("C01GNJRCQLD", slack.MsgOptionText(gm.ResultText(), false))
				sHandler.GameStorage.RemoveGame()
				continue
			}

			if gm.TurnPlayer().ID == "bot" {
				time.Sleep(time.Second * 2)
				botMove := gm.BotMove()
				fmt.Println("bot played: ", botMove)
				api.PostMessage("C01GNJRCQLD", slack.MsgOptionText("bot played", false))
				fmt.Println(gm)
			}

			if len(gm.Votes()) == 0 {
				continue
			}

			if gm.TurnPlayer().ID != "bot" {
				if time.Since(gm.LastMoveTime()) > 20*time.Second {
					fmt.Println("current votes: ", gm.Votes())
					err := gm.MoveTopVote()
					if err != nil {
						fmt.Println(err)
					}
					api.PostMessage("C01GNJRCQLD", slack.MsgOptionText("human moved", false))
					fmt.Println(gm)
				}
			}
		}
	}()

	fmt.Println("[INFO] Server listening")
	http.ListenAndServe(":5000", nil)
}
