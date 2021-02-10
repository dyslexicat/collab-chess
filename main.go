package main

import (
	"chess-slack/game"
	"chess-slack/handler"
	"chess-slack/rendering"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("error loading .env files")
	}

	// slack api bot token (xobx...)
	slackAuthToken := os.Getenv("SLACK_BOT_TOKEN")
	signingSecret := os.Getenv("SLACK_SIGNING_SECRET")
	hostname := os.Getenv("HOSTNAME")

	var gameStorage game.ChessStorage

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

	fmt.Println("[INFO] Server listening")
	http.ListenAndServe(":5000", nil)
}
