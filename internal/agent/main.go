package agent

import (
	"errors"
	"fmt"

	"github.com/a13labs/cobot/internal/algo"
	"github.com/a13labs/cobot/internal/nlp"
	"github.com/go-yaml/yaml"
)

type AgentStartArgs struct {
	StoragePath  string
	LogFile      string
	MinimumScore float64
	LLMHost      string
	LLMPort      int
	LLMModel     string
}

var DefaultArgs = AgentStartArgs{
	StoragePath:  "data",
	LogFile:      "",
	MinimumScore: 0.5,
	LLMHost:      "localhost",
	LLMPort:      11434,
	LLMModel:     "mistral",
}

type agentDef struct {
	Name            string `yaml:"name"`
	AllowReboot     bool   `yaml:"allow_reboot"`
	AllowPrivileged bool   `yaml:"allow_privileged"`
}

type AgentConfigFile struct {
	Agent   agentDef `yaml:"agent"`
	Actions []string `yaml:"actions"`
}

type AgentCtx struct {
	Storage       *Storage
	ActionDB      *ActionDB
	LLMClient     *nlp.LLMClient
	AgentCfg      AgentConfigFile
	UserArgs      AgentStartArgs
	WriterFunc    func(string) error
	InputChannel  chan string
	OutputChannel chan string
}

func NewAgentCtx(args *AgentStartArgs) (*AgentCtx, error) {

	ctx := &AgentCtx{
		UserArgs: *args,
	}

	// Initialize the logger
	var err error
	logger, err = NewLogger(args.LogFile)
	if err != nil {
		return nil, err
	}

	// Set the user arguments
	if ctx.UserArgs.StoragePath == "" {
		ctx.UserArgs.StoragePath = DefaultArgs.StoragePath
	}
	if ctx.UserArgs.MinimumScore == 0 {
		ctx.UserArgs.MinimumScore = DefaultArgs.MinimumScore
	}
	if ctx.UserArgs.LLMHost == "" {
		ctx.UserArgs.LLMHost = DefaultArgs.LLMHost
	}
	if ctx.UserArgs.LLMPort == 0 {
		ctx.UserArgs.LLMPort = DefaultArgs.LLMPort
	}
	if ctx.UserArgs.LLMModel == "" {
		ctx.UserArgs.LLMModel = DefaultArgs.LLMModel
	}

	// Initialize the storage
	ctx.Storage, err = NewStorage(ctx.UserArgs.StoragePath)
	if err != nil {
		return nil, errors.New("error initializing storage")
	}

	// Load the agent configuration
	// Check if the local folder has the required structure
	// If not, create the required structure
	_, err = ctx.Storage.Stat("agent-config.yaml")
	if err != nil {
		logger.Info("Agent configuration file not found. Creating a default agent configuration file.")
		ctx.AgentCfg = AgentConfigFile{
			Agent: agentDef{
				Name:            "default",
				AllowReboot:     false,
				AllowPrivileged: false,
			},
			Actions: []string{},
		}
		agentCfgData, err := yaml.Marshal(ctx.AgentCfg)
		if err != nil {
			logger.Error("Error marshalling agent configuration")
			return nil, errors.New("error marshalling agent configuration")
		}
		err = ctx.Storage.WriteFile("agent-config.yaml", agentCfgData, 0644)
		if err != nil {
			logger.Error("Error writing agent configuration file")
			return nil, errors.New("error writing agent configuration file")
		}
	} else {
		logger.Info("Loading agent configuration from storage")

		// Load the agent configuration file
		agentCfgData, err := ctx.Storage.ReadFile("agent-config.yaml")
		if err != nil {
			logger.Error("Error reading agent configuration file")
			return nil, errors.New("error reading agent configuration file")
		}

		// Unmarshal the agent configuration file
		if err := yaml.Unmarshal(agentCfgData, &ctx.AgentCfg); err != nil {
			logger.Error("Error parsing agent configuration file")
			return nil, errors.New("error parsing agent configuration file")
		}

		// Check if the agent configuration file has the required fields
		if ctx.AgentCfg.Agent.Name == "" {
			logger.Error("Agent name is empty")
			return nil, errors.New("agent name is empty")
		}

	}

	// Initialize the LLM client
	ctx.LLMClient = nlp.NewLLMClient(ctx.UserArgs.LLMHost, ctx.UserArgs.LLMPort, ctx.UserArgs.LLMModel)
	if ctx.LLMClient == nil {
		return nil, errors.New("error initializing LLM client")
	}

	// Initialize the action database
	ctx.ActionDB, err = NewActionDB(algo.StringList(ctx.AgentCfg.Actions), ctx.Storage, ctx.LLMClient)
	if err != nil {
		return nil, errors.New("error initializing action database")
	}

	ctx.WriterFunc = func(msg string) error {
		return nil
	}

	ctx.InputChannel = make(chan string)
	ctx.OutputChannel = make(chan string)

	go ctx.processInput()
	go ctx.processOutput()

	return ctx, nil
}

func (ctx *AgentCtx) SetWriterFunc(f func(string) error) {
	ctx.WriterFunc = f
}

func (ctx *AgentCtx) processInput() {

	for msg := range ctx.InputChannel {
		if msg == "exit" {
			break
		}
		ctx.process(msg)
	}

}

func (ctx *AgentCtx) processOutput() {
	for msg := range ctx.OutputChannel {
		if msg == "exit" {
			break
		}
		ctx.WriterFunc(msg)
	}
}

func (ctx *AgentCtx) process(userInput string) {

	isQuestion, err := isItAQuestion(ctx, userInput)
	if err != nil {
		logger.Error("Error parsing user input: %s", err)
		return
	}

	if isQuestion {
		ctx.Inform("Currently questions are not handled, only commands. No action will be taken.")
		return
	}

	validAction, err := isItemInList(ctx, userInput, ctx.ActionDB.GetActionDescriptions())
	if err != nil {
		logger.Error("Error parsing user input: %s", err)
		return
	}
	if validAction {

		actions, err := filterListItems(ctx, userInput, ctx.ActionDB.GetActionDescriptions())
		if err != nil {
			logger.Error("Error parsing user input: %s", err)
			return
		}

		if len(actions) == 0 {
			ctx.Inform("No actions were found. No action will be taken.")
			return
		}

		for _, action := range actions {
			logger.Info("Action: %s", ctx.ActionDB.ActionNames[action])
		}
	} else {
		ctx.Inform("No actions were found. No action will be taken.")
	}
}

func (ctx *AgentCtx) DispatchInput(userInput string) {
	ctx.InputChannel <- userInput
}

func (ctx *AgentCtx) GetAgentName() string {
	return ctx.AgentCfg.Agent.Name
}

func (ctx *AgentCtx) SayHello() {
	prompt := fmt.Sprintf("Your name is '%s'.You are polite.Inform the user you are ready to receive orders and greet him.", ctx.AgentCfg.Agent.Name)
	msg, err := generateAMessage(ctx, prompt)
	if err != nil {
		return
	}
	ctx.OutputChannel <- msg
}

func (ctx *AgentCtx) SayGoodBye() (string, error) {
	msg, err := generateAMessage(ctx, "Your name is '%s'.You are polite.Inform the user you are shutting down and say goodbye.")
	if err != nil {
		ctx.OutputChannel <- "error interacting with LLM"
	}
	return msg, nil
}

func (ctx *AgentCtx) Inform(text string) {
	prompt := fmt.Sprintf("Your name is '%s'.You are polite,inform the user,using your words,of the following event:'%s'.", ctx.AgentCfg.Agent.Name, text)
	msg, err := generateAMessage(ctx, prompt)
	if err != nil {
		ctx.OutputChannel <- "error interacting with LLM"
	}
	ctx.OutputChannel <- msg
}
