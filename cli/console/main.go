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
package console

import (
	"log"
	"os"

	"github.com/a13labs/infrabot/cli"
	"github.com/a13labs/infrabot/internal/agent"
	"github.com/a13labs/infrabot/internal/consoleBot"
	"github.com/spf13/cobra"
)

var configfile string
var logFile string

var agentName string
var agentLanguage string
var minimumScore float64

// telegramCmd represents the list command
var consoleCmd = &cobra.Command{
	Use:   "console",
	Short: "Recieve input from user argument",
	Long:  `Recieve all commands from argument.`,
	Run: func(cmd *cobra.Command, args []string) {
		if logFile != "" {
			logFile, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				log.Fatalf("Error opening log file: %v", err)
			}
			defer logFile.Close()
			log.SetOutput(logFile)
		}

		agent.Init(configfile)

		if agentName != "" {
			agent.OverrideAgentName(agentName)
		}

		if agentLanguage != "" {
			agent.OverrideAgentLanguage(agentLanguage)
		}

		if minimumScore > 0 {
			agent.OverrideMinimumScore(minimumScore)
		}

		consoleBot.Start()
		os.Exit(0)
	},
}

func init() {
	cli.RootCmd.AddCommand(consoleCmd)
	consoleCmd.Flags().StringVarP(&configfile, "definitions", "d", "agent.yaml", "Agent definiton file")
	consoleCmd.Flags().StringVarP(&logFile, "log", "l", "", "Log file")
	consoleCmd.Flags().StringVarP(&agentName, "agent", "a", "", "Agent name")
	consoleCmd.Flags().StringVarP(&agentLanguage, "language", "g", "", "Language")
	consoleCmd.Flags().Float64VarP(&minimumScore, "score", "r", 0.0, "Similarity minimum")
}
