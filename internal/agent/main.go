package agent

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/go-yaml/yaml"
	"github.com/kljensen/snowball"
	"gonum.org/v1/gonum/floats"
)

type agentConfig struct {
	Name            string `yaml:"name"`
	AllowReboot     bool   `yaml:"allow_reboot"`
	AllowPrivileged bool   `yaml:"allow_privileged"`
	Language        string `yaml:"language,omitempty"`
}

type actionDef struct {
	Description string   `yaml:"description"`
	Name        string   `yaml:"name"`
	Args        []string `yaml:"args,omitempty"`
	Plugin      string   `yaml:"plugin,omitempty"`
	Script      string   `yaml:"exec,omitempty"`
}

type configFile struct {
	Agent   agentConfig `yaml:"agent"`
	Actions []actionDef `yaml:"actions"`
}

type rankedAction struct {
	Action actionDef
	Score  float64
}

var agentCfg configFile
var agentLoaded = false
var minimumScore = 0.6

func RunAction(userInput string) (string, error) {

	if !agentLoaded {
		return "", errors.New("agent not loaded")
	}

	action, err := getAction(userInput)
	if err == nil {
		// Found a perfect match
		return fmt.Sprintf("Run action '%s'.\n", action.Name), nil
	}

	// No match, try to find by similarity
	actions, err := getRankedActions(userInput, agentCfg.Agent.Language)
	if err != nil {
		return "", err
	}

	if actions[0].Score < minimumScore {
		return fmt.Sprintf("No similar match for user input: '%s'.", userInput), nil
	}

	action = actions[0].Action
	return fmt.Sprintf("Run action '%s'.\n", action.Name), nil
}

func Init(file string) error {
	// Load the YAML file
	yamlFile, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("error reading file '%s'", file)
	}

	if err := yaml.Unmarshal(yamlFile, &agentCfg); err != nil {
		return fmt.Errorf("error parsing agent configuration file '%s'", file)
	}

	agentLoaded = true
	return nil
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

	return agentCfg.Agent.Language
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

func OverrideAgentLanguage(language string) {
	agentCfg.Agent.Language = language
}

func OverrideMinimumScore(score float64) {
	minimumScore = score
}

// Get action that matches the userInput
func getAction(userInput string) (actionDef, error) {

	if !agentLoaded {
		return actionDef{}, errors.New("not loaded")
	}

	for _, action := range agentCfg.Actions {

		if action.Name != userInput {
			continue
		}

		return action, nil
	}

	return actionDef{}, fmt.Errorf("action '%s' not found", userInput)
}

// Get ranked actions by similarity to the userInput
func getRankedActions(userInput string, language string) ([]rankedAction, error) {

	if !agentLoaded {
		return []rankedAction{}, errors.New("not loaded")
	}

	// Preprocess user input
	userInput = strings.ToLower(userInput)

	// Tokenize user input
	userTokens := tokenize(userInput, language)

	// Create TF-IDF vectors for user input and actions
	userVector := calculateTFIDFVector(userTokens, agentCfg.Actions, language)
	actionVectors := make([][]float64, len(agentCfg.Actions))
	for i, action := range agentCfg.Actions {
		actionTokens := tokenize(strings.ToLower(action.Description), language)
		actionVectors[i] = calculateTFIDFVector(actionTokens, agentCfg.Actions, language)
	}

	// Calculate cosine similarity
	similarityScores := make([]float64, len(agentCfg.Actions))
	for i, actionVector := range actionVectors {
		similarityScores[i] = cosineSimilarity(userVector, actionVector)
	}

	// Rank actions based on similarity scores
	rankedActions := rankActions(agentCfg.Actions, similarityScores)
	return rankedActions, nil
}

func tokenize(text string, language string) []string {
	tokens := strings.Fields(text)
	stemmedTokens := make([]string, len(tokens))
	for i, token := range tokens {
		stemmedToken, _ := snowball.Stem(token, language, false)
		stemmedTokens[i] = stemmedToken
	}
	return stemmedTokens
}

func calculateTFIDFVector(tokens []string, actions []actionDef, language string) []float64 {
	// Create a vocabulary of unique terms
	vocabulary := map[string]struct{}{}
	for _, action := range actions {
		actionTokens := tokenize(strings.ToLower(action.Description), language)
		for _, token := range actionTokens {
			vocabulary[token] = struct{}{}
		}
	}

	// Order the vocabulary
	var ordered_volcabulary []string
	for key := range vocabulary {
		ordered_volcabulary = append(ordered_volcabulary, key)
	}

	sort.Strings(ordered_volcabulary)

	// Create a TF-IDF vector
	vector := make([]float64, len(vocabulary))

	// Calculate the TF-IDF values for each term
	for i, term := range ordered_volcabulary {
		tf := float64(strings.Count(strings.ToLower(strings.Join(tokens, " ")), term))
		idf := inverseDocumentFrequency(term, actions)
		vector[i] = tf * idf
	}

	return vector
}

func inverseDocumentFrequency(term string, actions []actionDef) float64 {
	docCount := 0
	for _, action := range actions {
		if strings.Contains(strings.ToLower(action.Description), term) {
			docCount++
		}
	}
	if docCount == 0 {
		return 0
	}
	return float64(len(actions)) / float64(docCount)
}

func cosineSimilarity(vector1, vector2 []float64) float64 {
	dotProduct := floats.Dot(vector1, vector2)
	magnitude1 := floats.Norm(vector1, 2)
	magnitude2 := floats.Norm(vector2, 2)
	if magnitude1 == 0 || magnitude2 == 0 {
		return 0
	}
	return dotProduct / (magnitude1 * magnitude2)
}

func rankActions(actions []actionDef, similarityScores []float64) []rankedAction {
	rankedActions := make([]rankedAction, len(actions))
	for i, action := range actions {
		rankedActions[i] = rankedAction{
			Action: action,
			Score:  similarityScores[i],
		}
	}

	// Sort actions by similarity score in descending order
	for i := 0; i < len(rankedActions)-1; i++ {
		for j := i + 1; j < len(rankedActions); j++ {
			if rankedActions[i].Score < rankedActions[j].Score {
				rankedActions[i], rankedActions[j] = rankedActions[j], rankedActions[i]
			}
		}
	}

	return rankedActions
}
