/*
Copyright © 2023 Alexandre Pires

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package telegram

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/a13labs/cobot/cli"
	"github.com/a13labs/cobot/internal/agent"
	"github.com/a13labs/cobot/internal/telegramBot"
	"github.com/spf13/cobra"
)

var configfile string
var logFile string
var telegramToken string
var agentName string
var telegramChatId int64
var agentLanguage string
var minimumScore float64

// telegramCmd represents the list command
var telegramCmd = &cobra.Command{
	Use:   "telegram",
	Short: "Recieve input from a telegram channel",
	Long: `Recieve all commands from a telegram channel, make sure you
	provide a valid telegram token and a chat id.`,
	Run: func(cmd *cobra.Command, args []string) {
		if logFile != "" {
			logFile, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				log.Fatalf("Error opening log file: %v", err)
			}
			defer logFile.Close()
			log.SetOutput(logFile)
		}

		if err := agent.Init(configfile); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		if agent.GetAgentName() == "" && agentName == "" {
			value, exist := os.LookupEnv("BOT_AGENT_NAME")
			if !exist {
				fmt.Println("BOT_AGENT_NAME it's not defined, aborting.")
				os.Exit(1)
			}
			agentName = value
		}

		if agent.GetLanguage() == "" && agentLanguage == "" {
			value, exist := os.LookupEnv("BOT_AGENT_NAME")
			if !exist {
				fmt.Println("BOT_AGENT_NAME it's not defined, aborting.")
				os.Exit(1)
			}
			agentLanguage = value
		}

		if agentName != "" {
			agent.OverrideAgentName(agentName)
		}

		if agentLanguage != "" {
			agent.OverrideAgentLanguage(agentLanguage)
		}

		if minimumScore > 0 {
			agent.OverrideMinimumScore(minimumScore)
		}

		if telegramToken == "" {
			value, exist := os.LookupEnv("TELEGRAM_TOKEN")
			if !exist {
				fmt.Println("TELEGRAM_TOKEN it's not defined, aborting.")
				os.Exit(1)
			}
			telegramToken = value
		}

		if telegramChatId == 0 {
			valueStr, exist := os.LookupEnv("TELEGRAM_CHAT_ID")
			if !exist {
				fmt.Println("TELEGRAM_CHAT_ID it's not defined, aborting.")
				os.Exit(1)
			}
			value, err := strconv.ParseInt(valueStr, 10, 64)
			if err != nil {
				fmt.Println("TELEGRAM_CHAT_ID it's invalid, aborting.")
				os.Exit(1)
			}
			telegramChatId = value
		}

		telegramBot.Start(telegramToken, telegramChatId)
		os.Exit(0)
	},
}

func init() {
	cli.RootCmd.AddCommand(telegramCmd)
	telegramCmd.Flags().StringVarP(&configfile, "definitions", "d", "agent.yaml", "Agent definiton file")
	telegramCmd.Flags().StringVarP(&logFile, "log", "l", "", "Log file")
	telegramCmd.Flags().StringVarP(&agentName, "agent", "a", "", "Agent name")
	telegramCmd.Flags().StringVarP(&agentLanguage, "language", "g", "", "Language")
	telegramCmd.Flags().Float64VarP(&minimumScore, "score", "r", 0.0, "Similarity minimum")

	telegramCmd.Flags().StringVarP(&telegramToken, "token", "t", "", "Telegram bot token")
	telegramCmd.Flags().Int64VarP(&telegramChatId, "chat", "c", 0, "Telegram chat id")
}
