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
	Language     string
	MinimumScore float64
	LLMHost      string
	LLMPort      int
	LLLMModel    string
}

var agentLoaded = false

var DefaultArgs = AgentStartArgs{
	StoragePath:  "data",
	LogFile:      "",
	Language:     "english",
	MinimumScore: 0.5,
	LLMHost:      "localhost",
	LLMPort:      11434,
	LLLMModel:    "mistral",
}

type agentDef struct {
	Name            string `yaml:"name"`
	AllowReboot     bool   `yaml:"allow_reboot"`
	AllowPrivileged bool   `yaml:"allow_privileged"`
}

type agentConfig struct {
	Agent         agentDef               `yaml:"agent"`
	Actions       []string               `yaml:"actions"`
	KnowledgeBase map[string]interface{} `yaml:"knowledge_base"`
}

var agentStorage *Storage
var agentActionDB *ActionDB
var llmClient *nlp.LLMClient
var llmAgent *LLMAgent
var agentCfg agentConfig
var userArgs AgentStartArgs = DefaultArgs
var writerFunc func(string) error
var inputChannel chan string
var outputChannel chan string

func Init(args *AgentStartArgs) error {

	// Initialize the logger
	var err error
	logger, err = NewLogger(args.LogFile)
	if err != nil {
		return err
	}

	// Set the user arguments
	userArgs = *args
	if userArgs.StoragePath == "" {
		userArgs.StoragePath = DefaultArgs.StoragePath
	}
	if userArgs.Language == "" {
		userArgs.Language = DefaultArgs.Language
	}
	if userArgs.MinimumScore == 0 {
		userArgs.MinimumScore = DefaultArgs.MinimumScore
	}
	if userArgs.LLMHost == "" {
		userArgs.LLMHost = DefaultArgs.LLMHost
	}
	if userArgs.LLMPort == 0 {
		userArgs.LLMPort = DefaultArgs.LLMPort
	}
	if userArgs.LLLMModel == "" {
		userArgs.LLLMModel = DefaultArgs.LLLMModel
	}

	// Initialize the storage
	agentStorage, err = NewStorage(userArgs.StoragePath)
	if err != nil {
		return errors.New("error initializing storage")
	}

	// Load the agent configuration
	// Check if the local folder has the required structure
	// If not, create the required structure
	_, err = agentStorage.Stat("agent-config.yaml")
	if err != nil {
		logger.Info("Agent configuration file not found. Creating a default agent configuration file.")
		agentCfg := agentConfig{
			Agent: agentDef{
				Name:            "default",
				AllowReboot:     false,
				AllowPrivileged: false,
			},
			Actions:       []string{},
			KnowledgeBase: map[string]interface{}{},
		}
		agentCfgData, err := yaml.Marshal(agentCfg)
		if err != nil {
			logger.Error("Error marshalling agent configuration")
			return errors.New("error marshalling agent configuration")
		}
		err = agentStorage.WriteFile("agent-config.yaml", agentCfgData, 0644)
		if err != nil {
			logger.Error("Error writing agent configuration file")
			return errors.New("error writing agent configuration file")
		}
	} else {
		logger.Info("Loading agent configuration from storage")

		// Load the agent configuration file
		agentCfgData, err := agentStorage.ReadFile("agent-config.yaml")
		if err != nil {
			logger.Error("Error reading agent configuration file")
			return errors.New("error reading agent configuration file")
		}

		// Unmarshal the agent configuration file
		if err := yaml.Unmarshal(agentCfgData, &agentCfg); err != nil {
			logger.Error("Error parsing agent configuration file")
			return errors.New("error parsing agent configuration file")
		}

		// Check if the agent configuration file has the required fields
		if agentCfg.Agent.Name == "" {
			logger.Error("Agent name is empty")
			return errors.New("agent name is empty")
		}

	}

	// Initialize the LLM client
	llmClient = nlp.NewLLMClient(userArgs.LLMHost, userArgs.LLMPort, userArgs.LLLMModel)
	if llmClient == nil {
		return errors.New("error initializing LLM client")
	}

	// Initialize the action database
	agentActionDB, err = NewActionDB(algo.StringList(agentCfg.Actions), agentStorage, llmClient, userArgs.Language)
	if err != nil {
		return errors.New("error initializing action database")
	}

	err = agentActionDB.CacheInit()
	if err != nil {
		return errors.New("error building action database index")
	}

	writerFunc = func(msg string) error {
		return nil
	}

	inputChannel = make(chan string)
	outputChannel = make(chan string)

	go processInput()
	go processOutput()

	llmAgent = NewLLMAgent(llmClient, agentCfg.Agent.Name)

	agentLoaded = true
	return nil
}

func SetWriterFunc(f func(string) error) {
	writerFunc = f
}

func processInput() {

	for msg := range inputChannel {
		if msg == "exit" {
			break
		}
		process(msg)
	}

}

func processOutput() {
	for msg := range outputChannel {
		if msg == "exit" {
			break
		}
		writerFunc(msg)
	}
}

func process(userInput string) {

	actions, err := parseActions(userInput)
	if err != nil {
		logger.Error("Error parsing user input: %s", err)
		return
	}

	if len(actions) == 0 {
		Inform("No actions match the user input in the knowledge base. No action will be taken.")
		return
	}

	for _, action := range actions {
		logger.Info("Action: %s", agentActionDB.ActionNames[action])
	}
}

func DispatchInput(userInput string) {
	inputChannel <- userInput
}

func GetAgentName() string {
	return agentCfg.Agent.Name
}

func GetLanguage() string {
	return userArgs.Language
}

func SayHello() {
	msg, err := llmAgent.MessageRequest("Inform the user with your name and greet.")
	if err != nil {
		return
	}
	outputChannel <- msg
}

func SayGoodBye() (string, error) {
	msg, err := llmAgent.MessageRequest("Inform the user you are shutting down and say goodbye.")
	if err != nil {
		outputChannel <- "error interacting with LLM"
	}
	return msg, nil
}

func Inform(text string) {
	prompt := fmt.Sprintf("Inform the user of the following: '%s'", text)
	msg, err := llmAgent.MessageRequest(prompt)
	if err != nil {
		outputChannel <- "error interacting with LLM"
	}
	outputChannel <- msg
}

func parseActions(prompt string) ([]int, error) {

	availableActions := "Available actions:\n"
	for i, name := range agentActionDB.ActionNames {
		action := agentActionDB.Actions[name]
		availableActions += fmt.Sprintf("- ID: %d, Description: '%s'\n", i, action.Description)
	}
	instr := fmt.Sprintf("Consider the following action list:\n%s\nUser Input: %s\nParse the user input, provide the ID of all actions that match. Provide empty list if no action match.", availableActions, prompt)

	msg, err := llmAgent.IntListRequest(instr)
	if err != nil {
		return nil, err
	}
	return msg, nil
}
