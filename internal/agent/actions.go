package agent

import (
	"errors"

	"github.com/a13labs/cobot/internal/algo"
	"github.com/a13labs/cobot/internal/nlp"
	"github.com/go-yaml/yaml"
)

type ActionExecution struct {
	Plugin     string                 `yaml:"plugin,omitempty"`
	Parameters map[string]interface{} `yaml:"parameters"`
}

type Action struct {
	Description string          `yaml:"description"`
	Name        string          `yaml:"name"`
	Args        algo.StringList `yaml:"args,omitempty"`
	Exec        ActionExecution `yaml:"exec,omitempty"`
}

type ActionDB struct {
	LLMClient   *nlp.LLMClient
	Actions     map[string]Action
	ActionNames algo.StringList
	Driver      *Storage
}

func NewActionDB(actions algo.StringList, storage *Storage, llmClient *nlp.LLMClient) (*ActionDB, error) {

	_, err := storage.Stat("actions")
	if err != nil {
		storage.Mkdir("actions", 0755)
		logger.Info("Created actions folder, this folder should contain action files. Agent will start with an empty actions list.")
	}

	availableActions := make(map[string]Action)
	actionNames := algo.StringList{}
	for _, action := range actions {
		_, err := storage.Stat("actions/" + action + ".yaml")
		if err != nil {
			logger.Error("Action definition not found: " + action + ".yaml, skipping")
			continue
		}
		data, err := storage.ReadFile("actions/" + action + ".yaml")
		if err != nil {
			logger.Error("Error reading action file: " + action + ".yaml, skipping")
			continue
		}
		var a Action
		if err := yaml.Unmarshal(data, &a); err != nil {
			logger.Error("Error parsing action file: " + action + ".yaml, skipping")
			continue
		}

		availableActions[action] = a
		actionNames = append(actionNames, action)
	}

	return &ActionDB{
		Actions:     availableActions,
		ActionNames: actionNames,
		Driver:      storage,
		LLMClient:   llmClient,
	}, nil
}

func (adb *ActionDB) GetActions() map[string]Action {
	return adb.Actions
}

func (adb *ActionDB) GetActionNames() algo.StringList {
	return adb.ActionNames
}

func (adb *ActionDB) GetActionDescriptions() algo.StringList {
	descriptions := make(algo.StringList, len(adb.Actions))
	for i := 0; i < len(adb.ActionNames); i++ {
		actionName := adb.ActionNames[i]
		descriptions[i] = adb.Actions[actionName].Description
	}
	return descriptions
}

func (adb *ActionDB) GetAction(actionName string) (Action, error) {

	_, err := adb.Driver.Stat("actions/" + actionName + ".yaml")

	if err != nil {
		return Action{}, errors.New("action not found")
	}

	actionFile, err := adb.Driver.ReadFile("actions/" + actionName + ".yaml")
	if err != nil {
		return Action{}, errors.New("error reading action file")
	}

	var action Action
	if err := yaml.Unmarshal(actionFile, &action); err != nil {
		return Action{}, errors.New("error parsing action file")
	}

	return action, nil
}
