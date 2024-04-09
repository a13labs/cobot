package agent

import (
	"errors"
	"hash/crc32"
	"strings"

	"github.com/a13labs/cobot/internal/algo"
	"github.com/a13labs/cobot/internal/db"
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

type RankedAction struct {
	Action string
	Score  float64
}

type ActionDB struct {
	LLMClient   *nlp.LLMClient
	Actions     map[string]Action
	ActionNames algo.StringList
	Driver      *Storage
	Vocabulary  *nlp.Vocabulary
	VectorDB    *db.VectorDB
	Language    string
}

func NewActionDB(actions algo.StringList, storage *Storage, llmClient *nlp.LLMClient, language string) (*ActionDB, error) {

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
		Language:    language,
		LLMClient:   llmClient,
	}, nil
}

func (adb *ActionDB) GetActions() map[string]Action {
	return adb.Actions
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

func (adb *ActionDB) QueryDescription(description string, minimumScore float64) []string {

	// Create a query vector from the description
	description = strings.ToLower(description)
	tokens := adb.Vocabulary.Tokenize(description)
	query := adb.Vocabulary.CalculateTFIDFVector(tokens)
	indexes := adb.VectorDB.GetSimilarEntries(query, minimumScore)

	// Get the action names from the filtered indexes
	actions := make([]string, len(indexes))
	for i, index := range indexes {
		actionName := adb.ActionNames[index]
		actions[i] = adb.Actions[actionName].Description
	}
	return actions
}

func (adb *ActionDB) CacheInit() error {

	// Check if the cache folder exists
	_, err := adb.Driver.Stat("local/cache")
	if err != nil {
		logger.Info("Creating cache folder")
		adb.Driver.Mkdir("local/cache", 0755)
	}

	// Get the storage version to check if the cache already for the current version
	dbVersion, err := adb.Driver.GetVersion()
	if err != nil {
		logger.Error("Error getting storage version")
		return errors.New("error getting storage version")
	}

	indexFolder := "local/cache/" + dbVersion

	// Check if the cache folder exists if it does read data from it
	// otherwise create it
	_, err = adb.Driver.Stat(indexFolder)
	if err != nil {
		adb.Driver.Mkdir(indexFolder, 0755)
	}

	// If storage has local changes, check if the cache is still valid
	// by comparing the checksum of the action files
	if adb.Driver.HasLocalChanges() {

		changedFiles, err := adb.Driver.Status("actions/*.yaml")

		if err != nil {
			logger.Error("Error getting local changed actions files")
			return errors.New("error getting local changed actions files")
		}

		changedActions := algo.StringList{}
		for _, file := range changedFiles {
			action := strings.TrimSuffix(strings.TrimPrefix(file, "actions/"), ".yaml")
			if adb.ActionNames.Contains(action) {
				changedActions = append(changedActions, action)
			}
		}

		currentChecksum := adb.calculateChecksum(changedActions)
		checksumFile := indexFolder + "/actions.checksum"

		_, err = adb.Driver.Stat(checksumFile)
		invalidateCache := false
		if err != nil {
			logger.Info("Creating checksum file")
			data, err := adb.Driver.OpenFileStream(checksumFile)
			if err != nil {
				logger.Error("Error creating checksum file")
				return errors.New("error creating checksum file")
			}
			defer data.Close()
			data.WriteInt64(currentChecksum)
			invalidateCache = true
		} else {
			logger.Info("Loading cached checksum")
			data, err := adb.Driver.OpenFileStream(checksumFile)
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
			err = adb.Driver.RemoveAll("local/cache/" + dbVersion + "/" + adb.Language + ".vocabulary")
			if err != nil {
				logger.Error("Error removing vocabulary cache file")
			}
		}
	}

	// Create the Vocabulary and VectorDB objects
	vocabularyFilename := indexFolder + "/" + adb.Language + ".vocabulary"
	_, err = adb.Driver.Stat(vocabularyFilename)
	if err != nil {

		logger.Info("Creating vocabulary from action files")
		descriptions := make(algo.StringList, len(adb.Actions))
		for i := 0; i < len(adb.ActionNames); i++ {
			actionName := adb.ActionNames[i]
			descriptions[i] = adb.Actions[actionName].Description
		}

		adb.Vocabulary = nlp.NewVocabulary(descriptions, adb.Language)
		adb.VectorDB = db.NewVectorDB(len(adb.Vocabulary.Terms))

		// Calculate the TF-IDF vectors for each entry in the dataset
		adb.VectorDB.DataPoints = make([]db.DataPoint, len(adb.Actions))
		for i := 0; i < len(adb.Actions); i++ {
			tokens := adb.Vocabulary.Tokenize(strings.ToLower(descriptions[i]))
			adb.VectorDB.DataPoints[i] = db.DataPoint{ID: i, Data: adb.Vocabulary.CalculateTFIDFVector(tokens)}
		}

		dbFile, err := adb.Driver.OpenFileStream(vocabularyFilename)
		if err != nil {
			logger.Error("Error creating vocabulary file")
			return errors.New("error creating vocabulary file")
		}

		defer dbFile.Close()

		err = adb.Vocabulary.SaveToBinaryStream(dbFile)
		if err != nil {
			logger.Error("Error saving vocabulary file")
			return errors.New("error saving vocabulary file")
		}

		err = adb.VectorDB.SaveToBinaryStream(dbFile)
		if err != nil {
			logger.Error("Error saving vocabulary file")
			return errors.New("error saving vocabulary file")
		}

	} else {
		logger.Info("Loading cached vocabulary from cache folder")
		// Load cached vocabulary file in the binary format
		dbFile, err := adb.Driver.OpenFileStream(vocabularyFilename)
		if err != nil {
			logger.Error("Error creating vocabulary file")
			return errors.New("error creating vocabulary file")
		}

		defer dbFile.Close()

		adb.Vocabulary = nlp.NewVocabularyFromBinaryStream(dbFile, adb.Language)
		adb.VectorDB = db.NewVectorDBFromBinaryStream(dbFile)
	}

	return nil
}

func (adb *ActionDB) calculateChecksum(actionNames algo.StringList) int64 {

	logger := GetLogger()

	// Calculate the checksum of the action files
	checksums := algo.StringList{}

	// Calculate the checksum of the action files
	for _, action := range actionNames {
		fileData, err := adb.Driver.ReadFile("actions/" + action + ".yaml")
		if err != nil {
			logger.Error("Error reading action file: " + action + ".yaml")
			return 0
		}
		checksums = append(checksums, string(fileData))
	}

	return int64(crc32.ChecksumIEEE([]byte(strings.Join(checksums, ""))))
}
