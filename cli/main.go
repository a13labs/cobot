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
package cli

import (
	"fmt"
	"os"

	"github.com/a13labs/cobot/internal/agent"
	"github.com/spf13/cobra"
)

var logFile string
var language string
var minimumScore float64
var storagePath string
var llmHost string
var llmPort int
var llmModel string

var RootCmd = &cobra.Command{
	Use:   "cobot",
	Short: "A friendly customizable agent that can run actions on the local machine",
	Long: `A friendly customizable agent that can run actions on the local machine. The agent can run
	in two modes, in both modes the user can interact by writing commands.
	- console
	- telegram
	`,
}

func Execute() error {
	return RootCmd.Execute()
}

var AgentCtx *agent.AgentCtx

func init() {

	cobra.OnInitialize(initAgent)

	// Current working directory
	currDir, err := os.Getwd()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Set defaults storage path
	defaultPath := currDir + "/.data"

	RootCmd.PersistentFlags().StringVarP(&storagePath, "storage-path", "d", defaultPath, "Database path")
	RootCmd.PersistentFlags().StringVarP(&logFile, "log-file", "l", "", "Log file")
	RootCmd.PersistentFlags().StringVarP(&language, "language", "g", "english", "Language")
	RootCmd.PersistentFlags().Float64VarP(&minimumScore, "minimum-score", "r", 0.5, "Similarity minimum")
	RootCmd.PersistentFlags().StringVarP(&llmHost, "llm-host", "s", "localhost", "LLM host")
	RootCmd.PersistentFlags().IntVarP(&llmPort, "llm-port", "p", 11434, "LLM port")
	RootCmd.PersistentFlags().StringVarP(&llmModel, "llm-model", "m", "mistral", "LLM model")
}

func initAgent() {
	agentArgs := &agent.AgentStartArgs{
		StoragePath:  storagePath,
		LogFile:      logFile,
		MinimumScore: minimumScore,
		LLMHost:      llmHost,
		LLMPort:      llmPort,
		LLMModel:     llmModel,
	}
	var err error
	AgentCtx, err = agent.NewAgentCtx(agentArgs)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
