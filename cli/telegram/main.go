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
	"os"
	"strconv"

	"github.com/a13labs/cobot/cli"
	"github.com/a13labs/cobot/internal/agent"
	telegramChannel "github.com/a13labs/cobot/internal/channels/telegram"
	"github.com/spf13/cobra"
)

var telegramToken string
var telegramChatId int64

var logFile string
var language string
var minimumScore float64
var storagePath string

// telegramCmd represents the list command
var telegramCmd = &cobra.Command{
	Use:   "telegram",
	Short: "Recieve input from a telegram channel",
	Long: `Recieve all commands from a telegram channel, make sure you
	provide a valid telegram token and a chat id.`,
	Run: func(cmd *cobra.Command, args []string) {

		agentArgs := &agent.AgentStartArgs{
			StoragePath:  storagePath,
			LogFile:      logFile,
			Language:     language,
			MinimumScore: minimumScore,
		}

		if err := agent.Init(agentArgs); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
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

		telegramChannel.Start(telegramToken, telegramChatId)
		os.Exit(0)
	},
}

func init() {
	// Current working directory
	currDir, err := os.Getwd()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Set defautl storage path
	defaultPath := currDir + "/.data"

	cli.RootCmd.AddCommand(telegramCmd)
	telegramCmd.Flags().StringVarP(&storagePath, "storage", "d", defaultPath, "Database path")
	telegramCmd.Flags().StringVarP(&logFile, "log", "l", "", "Log file")
	telegramCmd.Flags().StringVarP(&language, "language", "g", "english", "Language")
	telegramCmd.Flags().Float64VarP(&minimumScore, "score", "r", 0.5, "Similarity minimum")

	telegramCmd.Flags().StringVarP(&telegramToken, "token", "t", "", "Telegram bot token")
	telegramCmd.Flags().Int64VarP(&telegramChatId, "chat", "c", 0, "Telegram chat id")
}
