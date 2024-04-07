package agent

/*
	A storage is required to store the agent configuration and actions.
	Under the hood, the storage will be a git repository. The storage will be
	initialized with the agent configuration file and the actions. The agent will
	load the configuration and actions from the storage.

	The folder structure of the storage will be as follows:
	- agent-config.yaml (agent configuration file)
	- actions/ (folder containing action files)
		- action1.yaml
		- action2.yaml
		- ...
	- plugins/ (folder containing plugin configuration files)
		- plugin1.yaml
		- plugin2.yaml
		- ...
	- local/ (folder containing local files, not stored in git)
		- logs/ (folder containing log files)
		- plugins/ (folder containing plugin binary files)
			- plugin1/
				- plugin1.so (plugin binary file)
				- resources/ (folder containing plugin resources)
			- plugin2/
				- plugin2.so (plugin binary file)
				- resources/ (folder containing plugin resources)
			- ...
		- cache/ (folder containing cache files)

	When initializing the storage, a path to an existing git repository must be provided.
	It is the responsibility of the caller to ensure that the git repository is properly
	initialized and configured. The agent will not create or configure the git repository.

	The agent will create the structure of the storage in the git repository, if it does not exist.

	The agent will load the configuration and actions from the storage.
*/

import (
	"errors"
	"hash/crc32"
	"os"
	"sort"
	"strings"

	"github.com/a13labs/cobot/internal/algo"
	"github.com/go-yaml/yaml"
	"github.com/kljensen/snowball"
	"gonum.org/v1/gonum/floats"
	"gopkg.in/src-d/go-git.v4"
)

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

type execDef struct {
	Plugin     string                 `yaml:"plugin,omitempty"`
	Parameters map[string]interface{} `yaml:"parameters"`
}

type actionDef struct {
	Description string   `yaml:"description"`
	Name        string   `yaml:"name"`
	Args        []string `yaml:"args,omitempty"`
	Exec        execDef  `yaml:"exec,omitempty"`
}

type rankedAction struct {
	Action string
	Score  float64
}

type termData struct {
	Token string
	IDF   float64
}

var dbPath string
var termsCache = []termData{}
var actionVectorsCache = map[string][]float64{}
var dbLanguage string
var agentCfg agentConfig
var minimumScore float64 = 0.5

// storageInit initializes the storage with the agent configuration and actions.
func storageInit(path string, language string) error {

	logger := GetLogger()

	// Check if the storage path is empty
	if path == "" {
		logger.Error("storage path is empty")
		return errors.New("storage path is empty")
	}

	// Check if the storage path is a valid git repository
	_, err := git.PlainOpen(path)
	if err != nil {
		logger.Error("storage path is not a valid git repository")
		return errors.New("storage path is not a valid git repository")
	}

	// Check if the local folder has the required structure
	// If not, create the required structure
	_, err = os.Stat(path + "/agent-config.yaml")
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
		err = os.WriteFile(path+"/agent-config.yaml", agentCfgData, 0644)
		if err != nil {
			logger.Error("Error writing agent configuration file")
			return errors.New("error writing agent configuration file")
		}
	} else {
		logger.Info("Loading agent configuration from storage")

		// Load the agent configuration file
		agentCfgData, err := os.ReadFile(path + "/agent-config.yaml")
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

	_, err = os.Stat(path + "/actions")
	if err != nil {
		os.Mkdir(path+"/actions", 0755)
		logger.Info("Created actions folder, this folder should contain action files. Agent will start with an empty actions list.")
	}

	_, err = os.Stat(path + "/plugins")
	if err != nil {
		os.Mkdir(path+"/plugins", 0755)
		logger.Info("Created plugins folder, this folder should contain plugin configuration files.")
	}

	_, err = os.Stat(path + "/local")
	if err != nil {
		os.Mkdir(path+"/local", 0755)
		logger.Info("Created local folder, this folder should contain local files.")
	}

	_, err = os.Stat(path + "/local/logs")
	if err != nil {
		os.Mkdir(path+"/local/logs", 0755)
		logger.Info("Created logs folder, this folder should contain log files.")
	}

	_, err = os.Stat(path + "/local/plugins")
	if err != nil {
		os.Mkdir(path+"/local/plugins", 0755)
		logger.Info("Created plugins folder, this folder should contain plugin binary files.")
	}

	_, err = os.Stat(path + "/local/cache")
	if err != nil {
		os.Mkdir(path+"/local/cache", 0755)
		logger.Info("Created cache folder, this folder should contain cache files.")
	}

	// Make sure .gitignore file exists and contains the required entries, if not create it.
	gitignore, err := os.ReadFile(path + "/.gitignore")
	if err != nil {
		logger.Info("Creating .gitignore file")
		gitignore = []byte("local/*\n")
		err = os.WriteFile(path+"/.gitignore", gitignore, 0644)
		if err != nil {
			logger.Error("Error writing .gitignore file")
			return errors.New("error writing .gitignore file")
		}
	} else {
		if !strings.Contains(string(gitignore), "local/*") {
			logger.Info("Updating .gitignore file")
			gitignore = append(gitignore, []byte("local/*\n")...)
			err = os.WriteFile(path+"/.gitignore", gitignore, 0644)
			if err != nil {
				logger.Error("Error writing .gitignore file")
				return errors.New("error writing .gitignore file")
			}
		}
	}

	logger.Info("storage initialized successfully")
	dbPath = path
	dbLanguage = language

	// Prepare the cache
	if err := storagePrepareCache(); err != nil {
		logger.Error("Error preparing cache")
		return errors.New("error preparing cache")
	}

	return nil
}

func storageSetMinimumScore(score float64) {
	minimumScore = score
}

func storagePrepareCache() error {

	if dbPath == "" {
		logger.Error("storage path is empty")
		return errors.New("storage path is empty")
	}

	// Check if the cache folder exists
	_, err := os.Stat(dbPath + "/local/cache")
	if err != nil {
		logger.Error("Cache folder not found")
		return errors.New("cache folder not found")
	}

	// Get the storage version to check if the cache already for the current version
	dbVersion, err := storageGetVersion()
	if err != nil {
		logger.Error("Error getting storage version")
		return errors.New("error getting storage version")
	}

	cacheFolder := dbPath + "/local/cache/" + dbVersion

	// Check if the cache folder exists if it does read data from it
	// otherwise create it
	_, err = os.Stat(cacheFolder)
	if err != nil {
		os.Mkdir(cacheFolder, 0755)
	}

	// If storage has local changes, check if the cache is still valid
	// by comparing the checksum of the action files
	if storageHasLocalChanges() {

		changedActions := storageGetChangedActions()
		currentChecksum := storageCalculateActionFilesChecksum(changedActions)

		checksumFile := cacheFolder + "/actions.checksum"
		_, err := os.Stat(checksumFile)
		invalidateCache := false
		if err != nil {
			logger.Info("Creating checksum file")
			data, err := algo.NewBinaryFileStream(checksumFile)
			if err != nil {
				logger.Error("Error creating checksum file")
				return errors.New("error creating checksum file")
			}
			defer data.Close()
			data.WriteInt64(currentChecksum)
			invalidateCache = true
		} else {
			logger.Info("Loading cached checksum")
			data, err := algo.NewBinaryFileStream(checksumFile)
			if err != nil {
				logger.Error("Error reading checksum file")
				return errors.New("error reading checksum file")
			}
			defer data.Close()

			previousChecksum, err := data.ReadInt64()
			if err != nil {
				logger.Error("Error reading checksum file")
				return errors.New("error reading checksum file")
			}

			invalidateCache = previousChecksum != currentChecksum
		}

		if invalidateCache {
			logger.Info("Local changes detected, invalidating cache")
			// Remove the cache folder and create a new one
			err = os.RemoveAll(dbPath + "/local/cache/" + dbVersion + "/" + dbLanguage + ".vocabulary")
			if err != nil {
				logger.Error("Error removing vocabulary cache file")
			}
		}
	}

	vocabularyFilename := cacheFolder + "/" + dbLanguage + ".vocabulary"
	_, err = os.Stat(vocabularyFilename)
	if err != nil {

		vocabulary := map[string]struct{}{}
		for _, action := range agentCfg.Actions {
			actionDef, err := storageLoadActionStorage(action)
			if err != nil {
				logger.Error("Error loading action definition for action '" + action + "', skipping")
				continue
			}
			lowerDesc := strings.ToLower(actionDef.Description)
			actionTokens := storageTokenize(lowerDesc, dbLanguage)
			for _, token := range actionTokens {
				vocabulary[token] = struct{}{}
			}
		}

		// Order the vocabulary
		terms := make([]string, 0, len(vocabulary))
		for key := range vocabulary {
			terms = append(terms, key)
		}
		sort.Strings(terms)

		// Calculate the IDF values for each term
		termsCache = make([]termData, len(terms))
		data := make([]termData, 0)
		for i, term := range terms {
			docCount := 0
			for _, action := range agentCfg.Actions {

				actionDef, err := storageLoadActionStorage(action)
				if err != nil {
					logger.Error("Error loading action definition for action '" + action + "', skipping")
					continue
				}
				lowerDesc := strings.ToLower(actionDef.Description)
				if strings.Contains(lowerDesc, term) {
					docCount++
				}
			}
			idf := 0.0
			if docCount != 0 {
				idf = float64(len(agentCfg.Actions)) / float64(docCount)
			}
			termsCache[i] = termData{Token: term, IDF: idf}
			data = append(data, termsCache[i])
		}

		actionVectorsCache = make(map[string][]float64, len(agentCfg.Actions))
		for _, action := range agentCfg.Actions {

			actionDef, err := storageLoadActionStorage(action)
			if err != nil {
				logger.Error("Error loading action definition for action '" + action + "', skipping")
				continue
			}
			descLower := strings.ToLower(actionDef.Description)
			actionTokens := storageTokenize(descLower, dbLanguage)
			actionVectorsCache[action] = storageCalculateTFIDFVector(actionTokens)
		}

		// Write the order vocabulary to a file, the vocabulary file is a binary file
		// containing the termData list

		logger.Info("Creating vocabulary file")
		vocabData, err := algo.NewBinaryFileStream(vocabularyFilename)
		if err != nil {
			logger.Error("Error creating vocabulary file")
			return errors.New("error creating vocabulary file")
		}
		defer vocabData.Close()

		// Marshal the data to the file
		// Write the number of terms to the file
		vocabData.WriteInt32(int32(len(data)))
		// Write the term data to the file
		for _, termData := range data {
			vocabData.WriteInt32(int32(len(termData.Token)))
			vocabData.WriteString(termData.Token)
			vocabData.WriteFloat64(termData.IDF)
		}
		// Write the number of actions to the file
		vocabData.WriteInt32(int32(len(actionVectorsCache)))
		// Write the action vectors to the file
		for action, vector := range actionVectorsCache {
			vocabData.WriteInt32(int32(len(action)))
			vocabData.WriteString(action)
			vocabData.WriteInt32(int32(len(vector)))
			for _, value := range vector {
				vocabData.WriteFloat64(value)
			}
		}

	} else {
		logger.Info("Loading cached vocabulary from cache folder")
		// Load cached vocabulary file in the binary format
		vocabularyFile, err := algo.NewBinaryFileStream(vocabularyFilename)
		if err != nil {
			logger.Error("Error reading vocabulary file")
			return errors.New("error reading vocabulary file")
		}
		defer vocabularyFile.Close()

		// Read the number of terms from the file
		numTerms, err := vocabularyFile.ReadInt32()
		if err != nil {
			logger.Error("Error reading vocabulary file")
			return errors.New("error reading vocabulary file")
		}

		termsCache = make([]termData, numTerms)
		for i := 0; i < int(numTerms); i++ {

			// Read the term data from the file
			termNameLength, err := vocabularyFile.ReadInt32()
			if err != nil {
				logger.Error("Error reading vocabulary file")
				return errors.New("error reading vocabulary file")
			}
			termName, err := vocabularyFile.ReadString(int(termNameLength))
			if err != nil {
				logger.Error("Error reading vocabulary file")
				return errors.New("error reading vocabulary file")
			}
			idf, err := vocabularyFile.ReadFloat64()
			if err != nil {
				logger.Error("Error reading vocabulary file")
				return errors.New("error reading vocabulary file")
			}
			termsCache[i] = termData{Token: termName, IDF: idf}
		}

		// Read the number of actions from the file
		numActions, err := vocabularyFile.ReadInt32()
		if err != nil {
			logger.Error("Error reading vocabulary file")
			return errors.New("error reading vocabulary file")
		}

		for i := 0; i < int(numActions); i++ {

			// Read the action name from the file
			actionNameLength, err := vocabularyFile.ReadInt32()
			if err != nil {
				logger.Error("Error reading vocabulary file")
				return errors.New("error reading vocabulary file")
			}
			actionName, err := vocabularyFile.ReadString(int(actionNameLength))
			if err != nil {
				logger.Error("Error reading vocabulary file")
				return errors.New("error reading vocabulary file")
			}

			// Read the action vector from the file
			vectorLength, err := vocabularyFile.ReadInt32()
			if err != nil {
				logger.Error("Error reading vocabulary file")
				return errors.New("error reading vocabulary file")
			}

			vector := make([]float64, vectorLength)
			for i := 0; i < int(vectorLength); i++ {
				value, err := vocabularyFile.ReadFloat64()
				if err != nil {
					logger.Error("Error reading vocabulary file")
					return errors.New("error reading vocabulary file")
				}
				vector[i] = value
			}

			// Check if the action definition exists in the storage
			_, err = os.Stat(dbPath + "/actions/" + actionName + ".yaml")
			if err != nil {
				logger.Error("Error loading action definition for action '" + actionName + "', skipping")
				continue
			}

			actionVectorsCache[actionName] = vector
		}
	}

	return nil
}

func storageLoadActionStorage(actionName string) (actionDef, error) {

	logger := GetLogger()

	// Check if the storage path is empty
	if dbPath == "" {
		logger.Error("storage path is empty")
		return actionDef{}, errors.New("storage path is empty")
	}

	_, err := os.Stat(dbPath + "/actions/" + actionName + ".yaml")
	if err != nil {
		logger.Error("Action file not found")
		return actionDef{}, errors.New("action file not found")
	}

	// Load the action file
	actionFile, err := os.ReadFile(dbPath + "/actions/" + actionName + ".yaml")
	if err != nil {
		logger.Error("Error reading action file")
		return actionDef{}, errors.New("error reading action file")
	}

	// Unmarshal the action file
	var action actionDef
	if err := yaml.Unmarshal(actionFile, &action); err != nil {
		logger.Error("Error parsing action file")
		return actionDef{}, errors.New("error parsing action file")
	}

	return action, nil
}

func storageGetRankedActions(userInput string) ([]rankedAction, error) {

	if !agentLoaded {
		return []rankedAction{}, errors.New("not loaded")
	}

	// Preprocess user input
	userInput = strings.ToLower(userInput)

	// Tokenize user input
	userTokens := storageTokenize(userInput, dbLanguage)

	// Create TF-IDF vectors for user input and actions
	userVector := storageCalculateTFIDFVector(userTokens)

	// Calculate the similarity score for each action
	rankedActions := make([]rankedAction, 0, len(actionVectorsCache))
	for action, actionVector := range actionVectorsCache {
		score := storageCosineSimilarity(userVector, actionVector)
		if score < minimumScore {
			continue
		}
		rankedActions = append(rankedActions, rankedAction{
			Action: action,
			Score:  score,
		})
	}

	// Sort actions by similarity score in descending order
	for i := 0; i < len(rankedActions)-1; i++ {
		for j := i + 1; j < len(rankedActions); j++ {
			if rankedActions[i].Score < rankedActions[j].Score {
				rankedActions[i], rankedActions[j] = rankedActions[j], rankedActions[i]
			}
		}
	}

	return rankedActions, nil
}

func storageGetVersion() (string, error) {

	logger := GetLogger()

	// Check if the storage path is empty
	if dbPath == "" {
		logger.Error("storage path is empty")
		return "", errors.New("storage path is empty")
	}

	// Get the git version
	repo, err := git.PlainOpen(dbPath)
	if err != nil {
		logger.Error("Error opening git repository")
		return "", errors.New("error opening git repository")
	}

	head, err := repo.Head()
	if err != nil {
		// If the repository is empty, return a default version
		logger.Info("Repository is empty, returning default version (v0.0.0)")
		return "v0.0.0", nil
	}

	return head.Hash().String(), nil
}

func storageHasLocalChanges() bool {

	logger := GetLogger()

	// Check if the storage path is empty
	if dbPath == "" {
		logger.Error("storage path is empty")
		return false
	}

	// Check if the git repository has local changes
	repo, err := git.PlainOpen(dbPath)
	if err != nil {
		logger.Error("Error opening git repository")
		return false
	}

	worktree, err := repo.Worktree()
	if err != nil {
		logger.Error("Error getting git worktree")
		return false
	}

	status, err := worktree.Status()
	if err != nil {
		logger.Error("Error getting git status")
		return false
	}

	return !status.IsClean()
}

func storageCalculateActionFilesChecksum(actionNames []string) int64 {

	logger := GetLogger()

	// Check if the storage path is empty
	if dbPath == "" {
		logger.Error("storage path is empty")
		return 0
	}

	// Calculate the checksum of the action files
	checksums := []string{}

	// Calculate the checksum of the action files
	for _, action := range actionNames {
		fileData, err := os.ReadFile(dbPath + "/actions/" + action + ".yaml")
		if err != nil {
			logger.Error("Error reading action file: " + action + ".yaml")
			return 0
		}
		checksums = append(checksums, string(fileData))
	}

	return int64(crc32.ChecksumIEEE([]byte(strings.Join(checksums, ""))))
}

func storageGetChangedActions() []string {

	logger := GetLogger()

	// Check if the storage path is empty
	if dbPath == "" {
		logger.Error("storage path is empty")
		return []string{}
	}

	// Check if the git repository has local changes
	repo, err := git.PlainOpen(dbPath)
	if err != nil {
		logger.Error("Error opening git repository")
		return []string{}
	}

	worktree, err := repo.Worktree()
	if err != nil {
		logger.Error("Error getting git worktree")
		return []string{}
	}

	status, err := worktree.Status()
	if err != nil {
		logger.Error("Error getting git status")
		return []string{}
	}

	changedActions := []string{}
	for file := range status {
		if strings.HasPrefix(file, "actions/") && strings.HasSuffix(file, ".yaml") {
			changedActions = append(changedActions, strings.TrimSuffix(strings.TrimPrefix(file, "actions/"), ".yaml"))
		}
	}

	return changedActions
}

func storageCalculateTFIDFVector(tokens []string) []float64 {

	logger := GetLogger()

	// Check if the storage path is empty
	if dbPath == "" {
		logger.Error("storage path is empty")
		return []float64{}
	}

	// Create a TF-IDF vector
	vector := make([]float64, len(termsCache))

	// Calculate the TF-IDF values for each term
	tokensStr := strings.ToLower(strings.Join(tokens, " "))
	for i, term := range termsCache {

		tf := float64(strings.Count(tokensStr, term.Token))
		idf := term.IDF
		vector[i] = tf * idf
	}

	return vector
}

// Calculate the cosine similarity between two vectors
func storageCosineSimilarity(vector1, vector2 []float64) float64 {
	dotProduct := floats.Dot(vector1, vector2)
	magnitude1 := floats.Norm(vector1, 2)
	magnitude2 := floats.Norm(vector2, 2)
	if magnitude1 == 0 || magnitude2 == 0 {
		return 0
	}
	return dotProduct / (magnitude1 * magnitude2)
}

// Tokenize the text and stem the tokens
func storageTokenize(text string, language string) []string {
	tokens := strings.Fields(text)
	stemmedTokens := make([]string, len(tokens))
	for i, token := range tokens {
		stemmedToken, _ := snowball.Stem(token, language, false)
		stemmedTokens[i] = stemmedToken
	}
	return stemmedTokens
}
