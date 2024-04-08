package agent

import (
	"errors"
	"fmt"

	"github.com/a13labs/cobot/internal/algo"
	"github.com/go-yaml/yaml"
)

type AgentStartArgs struct {
	StoragePath  string
	LogFile      string
	Language     string
	MinimumScore float64
}

var agentLoaded = false

var DefaultArgs = AgentStartArgs{
	StoragePath:  "data",
	LogFile:      "",
	Language:     "english",
	MinimumScore: 0.5,
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
var agentCfg agentConfig
var userArgs AgentStartArgs = DefaultArgs

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

	// Initialize the action database
	agentActionDB, err = NewActionDB(algo.StringList(agentCfg.Actions), agentStorage, userArgs.Language)
	if err != nil {
		return errors.New("error initializing action database")
	}

	err = agentActionDB.CacheInit()
	if err != nil {
		return errors.New("error building action database index")
	}

	agentLoaded = true
	return nil
}

func RunAction(userInput string) (string, error) {

	if !agentLoaded {
		return "", errors.New("agent not loaded")
	}

	actions := agentActionDB.QueryDescription(userInput, userArgs.MinimumScore)

	if len(actions) == 0 {
		return fmt.Sprintf("No match for user input: '%s'.", userInput), nil
	}

	for _, action := range actions {
		logger.Info("Action: %s", action)
	}

	return fmt.Sprintf("Found %d possible actions.", len(actions)), nil
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
