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
	"io/fs"
	"os"

	"github.com/a13labs/cobot/internal/algo"
	"github.com/a13labs/cobot/internal/io"
	"gopkg.in/src-d/go-git.v4"
)

type Storage struct {
	localPath string
}

func NewStorage(path string) (*Storage, error) {

	logger := GetLogger()

	// Check if the storage path is a valid git repository
	_, err :=
		git.PlainOpen(path)
	if err != nil {
		logger.Error("storage path is not a valid git repository")
		return nil, errors.New("storage path is not a valid git repository")
	}

	return &Storage{
		localPath: path,
	}, nil
}

func (s *Storage) Stat(path string) (os.FileInfo, error) {

	logger := GetLogger()

	// Check if the sub path is empty
	if path == "" {
		logger.Error("sub path is empty")
		return nil, errors.New("sub path is empty")
	}

	// Get the file info
	fileInfo, err := os.Stat(s.localPath + "/" + path)
	if err != nil {
		logger.Error("Error getting file info for path " + path)
		return nil, err
	}

	return fileInfo, nil
}

func (s *Storage) ReadFile(path string) ([]byte, error) {

	logger := GetLogger()

	// Check if the sub path is empty
	if path == "" {
		logger.Error("sub path is empty")
		return nil, errors.New("sub path is empty")
	}

	// Read the file
	data, err := os.ReadFile(s.localPath + "/" + path)
	if err != nil {
		logger.Error("Error reading file")
		return nil, err
	}

	return data, nil
}

func (s *Storage) WriteFile(path string, data []byte, perm fs.FileMode) error {

	logger := GetLogger()

	// Check if the sub path is empty
	if path == "" {
		logger.Error("sub path is empty")
		return errors.New("sub path is empty")
	}

	// Write the file
	err := os.WriteFile(s.localPath+"/"+path, data, perm)
	if err != nil {
		logger.Error("Error writing file")
		return err
	}

	return nil
}

func (s *Storage) OpenFileStream(path string) (*io.BinaryFileStream, error) {

	logger := GetLogger()

	// Check if the sub path is empty
	if path == "" {
		logger.Error("sub path is empty")
		return nil, errors.New("sub path is empty")
	}

	// Create a new binary file stream
	stream, err := io.NewBinaryFileStream(s.localPath + "/" + path)
	if err != nil {
		logger.Error("Error creating binary file stream")
		return nil, err
	}

	return stream, nil

}

func (s *Storage) RemoveFile(path string) error {

	logger := GetLogger()

	// Check if the sub path is empty
	if path == "" {
		logger.Error("sub path is empty")
		return errors.New("sub path is empty")
	}

	// Remove the file
	err := os.Remove(s.localPath + "/" + path)
	if err != nil {
		logger.Error("Error removing file")
		return err
	}

	return nil
}

func (s *Storage) ListFiles(path string) ([]string, error) {

	logger := GetLogger()

	// Check if the sub path is empty
	if path == "" {
		logger.Error("sub path is empty")
		return nil, errors.New("sub path is empty")
	}

	// List the files
	files, err := os.ReadDir(s.localPath + "/" + path)
	if err != nil {
		logger.Error("Error listing files")
		return nil, err
	}

	var fileNames []string
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}

	return fileNames, nil
}

func (s *Storage) MkdirAll(path string, perm fs.FileMode) error {

	logger := GetLogger()

	// Check if the sub path is empty
	if path == "" {
		logger.Error("sub path is empty")
		return errors.New("sub path is empty")
	}

	// Create the directory
	err := os.MkdirAll(s.localPath+"/"+path, perm)
	if err != nil {
		logger.Error("Error creating directory")
		return err
	}

	return nil
}

func (s *Storage) Remove(path string) error {

	logger := GetLogger()

	// Check if the sub path is empty
	if path == "" {
		logger.Error("sub path is empty")
		return errors.New("sub path is empty")
	}

	// Remove the file
	err := os.Remove(s.localPath + "/" + path)
	if err != nil {
		logger.Error("Error removing file")
		return err
	}

	return nil
}

func (s *Storage) RemoveAll(path string) error {

	logger := GetLogger()

	// Check if the sub path is empty
	if path == "" {
		logger.Error("sub path is empty")
		return errors.New("sub path is empty")
	}

	// Remove the directory
	err := os.RemoveAll(s.localPath + "/" + path)
	if err != nil {
		logger.Error("Error removing directory")
		return err
	}

	return nil
}

func (s *Storage) Mkdir(path string, perm fs.FileMode) error {

	logger := GetLogger()

	// Check if the sub path is empty
	if path == "" {
		logger.Error("sub path is empty")
		return errors.New("sub path is empty")
	}

	// Create the directory
	err := os.Mkdir(s.localPath+"/"+path, perm)
	if err != nil {
		logger.Error("Error creating directory")
		return err
	}

	return nil
}

func (s *Storage) HasLocalChanges() bool {

	logger := GetLogger()

	// Check if the git repository has local changes
	repo, err := git.PlainOpen(s.localPath)
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

func (s *Storage) Status(wildcard string) ([]string, error) {

	logger := GetLogger()

	// Check if the git repository has local changes
	repo, err := git.PlainOpen(s.localPath)
	if err != nil {
		logger.Error("Error opening git repository")
		return nil, err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		logger.Error("Error getting git worktree")
		return nil, err
	}

	status, err := worktree.Status()
	if err != nil {
		logger.Error("Error getting git status")
		return nil, err
	}

	var changedFiles []string
	for file := range status {
		if algo.Match(wildcard, file) {
			changedFiles = append(changedFiles, file)
		}
	}

	return changedFiles, nil
}

func (s *Storage) GetVersion() (string, error) {

	logger := GetLogger()

	// Get the git version
	repo, err := git.PlainOpen(s.localPath)
	if err != nil {
		logger.Error("Error opening git repository")
		return "", err
	}

	head, err := repo.Head()
	if err != nil {
		// If the repository is empty, return a default version
		logger.Info("Repository is empty, returning default version (v0.0.0)")
		return "v0.0.0", nil
	}

	return head.Hash().String(), nil
}
