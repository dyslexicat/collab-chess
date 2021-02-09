package main

import (
	"chess-slack/game"
	"chess-slack/handler"
	"chess-slack/rendering"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/nlopes/slack"
	"github.com/notnil/chess"
)

var colorToHex = map[game.Color]string{
	game.Black: "#000000",
	game.White: "#eeeeee",
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("error loading .env files")
	}

	// slack api bot token (xobx...)
	slackAuthToken := os.Getenv("SLACK_BOT_TOKEN")
	signingSecret := os.Getenv("SLACK_SIGNING_SECRET")
	hostname := os.Getenv("TEST_HOSTNAME")

	var gameStorage game.ChessStorage
	api := slack.New(slackAuthToken)

	memoryStore := game.NewMemoryStore()
	gameStorage = memoryStore

	renderLink := rendering.NewRenderLink(hostname, signingSecret)

	sHandler := handler.SlackHandler{
		SigningKey:   signingSecret,
		BotToken:     slackAuthToken,
		GameStorage:  gameStorage,
		LinkRenderer: renderLink,
	}

	http.Handle("/slack/events", sHandler)

	http.Handle("/board", rendering.BoardRenderHandler{
		LinkRenderer: renderLink,
	})
	http.Handle("/board.png", rendering.BoardRenderHandler{
		LinkRenderer: renderLink,
	})

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

				link, _ := sHandler.LinkRenderer.CreateLink(gm)

				boardAttachment := slack.Attachment{
					Text:     botMove.String(),
					ImageURL: link.String(),
					Color:    colorToHex[gm.Turn()],
				}

				api.PostMessage("C01GNJRCQLD", slack.MsgOptionText("bot played", false), slack.MsgOptionAttachments(boardAttachment))
				fmt.Println(gm)
			}

			if gm.TurnPlayer().ID != "bot" {
				if time.Since(gm.LastMoveTime()) > 30*time.Second {
					fmt.Println("removing the current game from pool")
					sHandler.GameStorage.RemoveGame()
					g, _ := sHandler.GameStorage.RetrieveGame()
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
	}()

	fmt.Println("[INFO] Server listening")
	http.ListenAndServe(":5000", nil)
}
