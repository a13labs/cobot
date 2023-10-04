package telegramBot

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/a13labs/infrabot/internal/agent"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var shutdownSignal = make(chan os.Signal, 1)

// StartBot initializes and starts the Telegram bot with a channel listener.
func Start(token string, chatId int64) {

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("Error initializing Telegram bot: %v", err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatalf("Error creating update channel: %v", err)
	}

	// sets up a signal handler to listen for SIGQUIT and SIGINT.
	signal.Notify(shutdownSignal, syscall.SIGQUIT)
	signal.Notify(shutdownSignal, syscall.SIGINT)

	msg := tgbotapi.NewMessage(chatId, agent.SayHello())
	bot.Send(msg)

	// Listen for messages in the channel
	for {
		select {
		case update := <-updates:
			if update.Message == nil {
				continue
			}

			if update.Message.Chat.ID != chatId {
				continue
			}

			// Handle messages from the specified channel
			userInput := update.Message.Text
			if strings.HasPrefix(userInput, "@") {
				tokens := strings.Split(userInput, " ")
				if len(tokens) > 0 {

					// Convert the string to a rune slice
					runes := []rune(tokens[0])

					if len(runes) > 0 {
						targetAgent := string(runes[1:])
						txt := "How can I help you?"

						if targetAgent == agent.GetAgentName() {

							if len(tokens) > 1 {

								txt, err = agent.RunAction(userInput)

								if err != nil {
									txt = fmt.Sprintf("Error: %s", err.Error())
								}
							}

							msg := tgbotapi.NewMessage(update.Message.Chat.ID, txt)
							msg.ReplyToMessageID = update.Message.MessageID
							bot.Send(msg)
						}
					}
				}
			}
		case <-shutdownSignal:
			msg := tgbotapi.NewMessage(chatId, agent.SayGoodBye())
			bot.Send(msg)
			return
		}
	}
}
