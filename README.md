# @chess-bot

♟️Hey, I live in __#playchess__ at the [Hack Club Slack](https://slack.hackclub.com). I am a chess bot that is aiming to provide a _Humans vs. AI_ experience. Each turn, users in the channel can vote on a move and after forty seconds the top voted move gets played. Let's try to beat Stockfish as a collaborative effort!

Built on top of [CJSaylor](https://github.com/cjsaylor/chessbot)'s amazing @chessbot

If there is no active game for 3 minutes, then it is removed from memory.

#### COMMANDS
```
!start (white/black - optional) - starts a new game
!move [notation] - Votes on the specified move. For example, !move e4 or !move Nc6. Each turn top voted move gets played.
!board - Shows the current state of the chess board
```

#### SETUP
- Create a new Slack App and add the following bot token scopes from "OAuth & Permissions": *app_mentions:read*, *channels:history*, *chat:write*
- Go to "Event Subscriptions", enable events and subscribe to the *message.channels* and *app_mention* events
- Install the app to your Workspace from the "OAuth & Permissions" page, grab your "Bot User OAuth Access Token" and set it as the SLACK_BOT_TOKEN in your environment
- Under "Basic Information", grab the Signing Secret and set it as SLACK_SIGNING_SECRET in your environment
- Set the CHANNEL_ID (the channel you want the bot to be active) and APP_HOSTNAME (the public url where you will be listening for slack events) variables in your environment
- For local development you need to place the relevant stockfish binary for your OS in a folder in your PATH
- If you are developing locally, use ngrok to create a public url and put "{your_ngrok_url}/slack/events" to the "Request URL" under "Event Subscriptions"
- **!!!** If you are deploying using the Dockerfile or you are on a Linux system you have to install the MS fonts to see the ranks and files on the board image using

```
apt-get install ttf-mscorefonts-installer fontconfig && fc-cache -f
``` 

You can confirm if the fonts are installed correctly by using "fc-match Arial" 

#### ISSUES
- Sometimes the ranks and files are not rendered properly on the board.

#### IDEAS
- Instead of Stockfish create a Chess engine from scratch?
- Persist games in a database so that we can see who played how many games and detailed statistics?
- Let users know when it is the last 10-15 seconds to make a move?
- !votes command to show which moves have been voted so far?
