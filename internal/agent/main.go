package agent

import (
	"errors"
	"fmt"
)

type AgentStartArgs struct {
	StoragePath  string
	LogFile      string
	Language     string
	MinimumScore float64
}

var agentLoaded = false
var defaultArgs = AgentStartArgs{
	StoragePath:  "data",
	LogFile:      "",
	Language:     "english",
	MinimumScore: 0.5,
}
var userArgs AgentStartArgs

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
		userArgs.StoragePath = defaultArgs.StoragePath
	}
	if userArgs.Language == "" {
		userArgs.Language = defaultArgs.Language
	}
	if userArgs.MinimumScore == 0 {
		userArgs.MinimumScore = defaultArgs.MinimumScore
	}

	// Initialize the storage
	if err := storageInit(userArgs.StoragePath, userArgs.Language); err != nil {
		return errors.New("error initializing storage")
	}

	storageSetMinimumScore(userArgs.MinimumScore)

	agentLoaded = true
	return nil
}

func RunAction(userInput string) (string, error) {

	if !agentLoaded {
		return "", errors.New("agent not loaded")
	}

	actions, err := storageGetRankedActions(userInput)

	if err != nil {
		return "", err
	}

	if len(actions) == 0 {
		return fmt.Sprintf("No match for user input: '%s'.", userInput), nil
	}

	if actions[0].Score < userArgs.MinimumScore {
		return fmt.Sprintf("No similar match for user input: '%s'.", userInput), nil
	}

	return fmt.Sprintf("Run action '%s'.\n", actions[0].Action), nil
}

func GetAgentName() string {
	if !agentLoaded {
		return ""
	}

	return agentCfg.Agent.Name
}

func GetLanguage() string {
	if !agentLoaded {
		return ""
	}

	return userArgs.Language
}

func SayHello() string {
	return fmt.Sprintf("Hello! Agent %s ready.", agentCfg.Agent.Name)
}

func SayGoodBye() string {
	return fmt.Sprintf("Bye! Agent %s shutting down.", agentCfg.Agent.Name)
}

func OverrideAgentName(name string) {
	agentCfg.Agent.Name = name
}
