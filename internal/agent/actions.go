package agent

import (
	"errors"
	"hash/crc32"
	"strings"

	"github.com/a13labs/cobot/internal/algo"
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
	Actions    algo.StringList
	Driver     *Storage
	Vocabulary *algo.Vocabulary
	VectorDB   *algo.VectorDB
	Language   string
}

func NewActionDB(actions algo.StringList, storage *Storage, language string) (*ActionDB, error) {

	_, err := storage.Stat("actions")
	if err != nil {
		storage.Mkdir("actions", 0755)
		logger.Info("Created actions folder, this folder should contain action files. Agent will start with an empty actions list.")
	}

	availableActions := algo.StringList{}
	for _, action := range actions {
		_, err := storage.Stat("actions/" + action + ".yaml")
		if err != nil {
			logger.Error("Action definition not found: " + action + ".yaml, skipping")
			continue
		}
		availableActions = append(availableActions, action)
	}

	return &ActionDB{
		Actions:  availableActions,
		Driver:   storage,
		Language: language,
	}, nil
}

func (db *ActionDB) GetActions() algo.StringList {
	return db.Actions
}

func (db *ActionDB) GetAction(actionName string) (Action, error) {

	_, err := db.Driver.Stat("actions/" + actionName + ".yaml")

	if err != nil {
		return Action{}, errors.New("action not found")
	}

	actionFile, err := db.Driver.ReadFile("actions/" + actionName + ".yaml")
	if err != nil {
		return Action{}, errors.New("error reading action file")
	}

	var action Action
	if err := yaml.Unmarshal(actionFile, &action); err != nil {
		return Action{}, errors.New("error parsing action file")
	}

	return action, nil
}

func (db *ActionDB) AddAction(action Action, overwrite bool) error {

	_, err := db.Driver.Stat("actions/" + action.Name + ".yaml")

	if err == nil && !overwrite {
		return errors.New("action already exists")
	}

	actionFile, err := yaml.Marshal(action)
	if err != nil {
		return errors.New("error marshalling action")
	}

	err = db.Driver.WriteFile("actions/"+action.Name+".yaml", actionFile, 0644)
	if err != nil {
		return errors.New("error writing action file")
	}

	db.Actions.Add(action.Name)

	return nil
}

func (db *ActionDB) RemoveAction(actionName string) error {

	_, err := db.Driver.Stat("actions/" + actionName + ".yaml")
	if err != nil {
		return errors.New("action not found")
	}

	err = db.Driver.Remove("actions/" + actionName + ".yaml")
	if err != nil {
		return errors.New("error removing action file")
	}

	db.Actions.Remove(actionName)

	return nil
}

func (db *ActionDB) UpdateAction(action Action) error {

	_, err := db.Driver.Stat("actions/" + action.Name + ".yaml")
	if err != nil {
		return errors.New("action not found")
	}

	actionFile, err := yaml.Marshal(action)
	if err != nil {
		return errors.New("error marshalling action")
	}

	err = db.Driver.WriteFile("actions/"+action.Name+".yaml", actionFile, 0644)
	if err != nil {
		return errors.New("error writing action file")
	}

	return nil
}

func (db *ActionDB) QueryDescription(description string, minimumScore float64) []string {

	// Create a query vector from the description
	description = strings.ToLower(description)
	tokens := db.Vocabulary.Tokenize(description)
	query := db.Vocabulary.CalculateTFIDFVector(tokens)
	indexes := db.VectorDB.GetSimilarEntries(query, minimumScore)

	// Get the action names from the filtered indexes
	actions := make([]string, len(indexes))
	for i, index := range indexes {
		actions[i] = db.Actions[index]
	}
	return actions
}

func (db *ActionDB) CacheInit() error {

	// Check if the cache folder exists
	_, err := db.Driver.Stat("local/cache")
	if err != nil {
		logger.Info("Creating cache folder")
		db.Driver.Mkdir("local/cache", 0755)
	}

	// Get the storage version to check if the cache already for the current version
	dbVersion, err := db.Driver.GetVersion()
	if err != nil {
		logger.Error("Error getting storage version")
		return errors.New("error getting storage version")
	}

	indexFolder := "local/cache/" + dbVersion

	// Check if the cache folder exists if it does read data from it
	// otherwise create it
	_, err = db.Driver.Stat(indexFolder)
	if err != nil {
		db.Driver.Mkdir(indexFolder, 0755)
	}

	// If storage has local changes, check if the cache is still valid
	// by comparing the checksum of the action files
	if db.Driver.HasLocalChanges() {

		changedFiles, err := db.Driver.Status("actions/*.yaml")

		if err != nil {
			logger.Error("Error getting local changed actions files")
			return errors.New("error getting local changed actions files")
		}

		changedActions := algo.StringList{}
		for _, file := range changedFiles {
			action := strings.TrimSuffix(strings.TrimPrefix(file, "actions/"), ".yaml")
			if db.Actions.Contains(action) {
				changedActions = append(changedActions, action)
			}
		}

		currentChecksum := db.calculateChecksum(changedActions)
		checksumFile := indexFolder + "/actions.checksum"

		_, err = db.Driver.Stat(checksumFile)
		invalidateCache := false
		if err != nil {
			logger.Info("Creating checksum file")
			data, err := db.Driver.OpenFileStream(checksumFile)
			if err != nil {
				logger.Error("Error creating checksum file")
				return errors.New("error creating checksum file")
			}
			defer data.Close()
			data.WriteInt64(currentChecksum)
			invalidateCache = true
		} else {
			logger.Info("Loading cached checksum")
			data, err := db.Driver.OpenFileStream(checksumFile)
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
			err = db.Driver.RemoveAll("local/cache/" + dbVersion + "/" + db.Language + ".vocabulary")
			if err != nil {
				logger.Error("Error removing vocabulary cache file")
			}
		}
	}

	// Create the Vocabulary and VectorDB objects
	vocabularyFilename := indexFolder + "/" + db.Language + ".vocabulary"
	_, err = db.Driver.Stat(vocabularyFilename)
	if err != nil {

		logger.Info("Creating vocabulary from action files")
		descriptions := make(algo.StringList, len(db.Actions))
		for i := 0; i < len(db.Actions); i++ {
			data, err := db.GetAction(db.Actions[i])
			if err != nil {
				logger.Error("Error reading action file: " + db.Actions[i] + ".yaml")
				return errors.New("error reading action file")
			}
			descriptions[i] = data.Description
		}

		db.Vocabulary = algo.NewVocabulary(descriptions, db.Language)
		db.VectorDB = algo.NewVectorDB(len(db.Vocabulary.Terms))

		// Calculate the TF-IDF vectors for each entry in the dataset
		db.VectorDB.DataPoints = make([]algo.DataPoint, len(db.Actions))
		for i := 0; i < len(db.Actions); i++ {
			tokens := db.Vocabulary.Tokenize(strings.ToLower(descriptions[i]))
			db.VectorDB.DataPoints[i] = algo.DataPoint{ID: i, Data: db.Vocabulary.CalculateTFIDFVector(tokens)}
		}

		dbFile, err := db.Driver.OpenFileStream(vocabularyFilename)
		if err != nil {
			logger.Error("Error creating vocabulary file")
			return errors.New("error creating vocabulary file")
		}

		defer dbFile.Close()

		err = db.Vocabulary.SaveToBinaryStream(dbFile)
		if err != nil {
			logger.Error("Error saving vocabulary file")
			return errors.New("error saving vocabulary file")
		}

		err = db.VectorDB.SaveToBinaryStream(dbFile)
		if err != nil {
			logger.Error("Error saving vocabulary file")
			return errors.New("error saving vocabulary file")
		}

	} else {
		logger.Info("Loading cached vocabulary from cache folder")
		// Load cached vocabulary file in the binary format
		dbFile, err := db.Driver.OpenFileStream(vocabularyFilename)
		if err != nil {
			logger.Error("Error creating vocabulary file")
			return errors.New("error creating vocabulary file")
		}

		defer dbFile.Close()

		db.Vocabulary = algo.NewVocabularyFromBinaryStream(dbFile, db.Language)
		db.VectorDB = algo.NewVectorDBFromBinaryStream(dbFile)
	}

	return nil
}

func (db *ActionDB) calculateChecksum(actionNames algo.StringList) int64 {

	logger := GetLogger()

	// Calculate the checksum of the action files
	checksums := algo.StringList{}

	// Calculate the checksum of the action files
	for _, action := range actionNames {
		fileData, err := db.Driver.ReadFile("actions/" + action + ".yaml")
		if err != nil {
			logger.Error("Error reading action file: " + action + ".yaml")
			return 0
		}
		checksums = append(checksums, string(fileData))
	}

	return int64(crc32.ChecksumIEEE([]byte(strings.Join(checksums, ""))))
}
