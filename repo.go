package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

func addRepo(value string) (validationMsg string, err error) {
	if value == "" {
		return "must provide a value", nil
	}
	if !isRepoValid(value) {
		return fmt.Sprintf("%s is not valid, you must provide a valid repo ssh clone url (e.g., git@github.com:JeremiahVaughan/strength-gadget-v5.git)", value), nil
	}

	var wg sync.WaitGroup
	errChan := make(chan error, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()
		var e error
		e = cloneRepo(value)
		if e != nil {
			errChan <- fmt.Errorf("error, when cloneRepo() for addRepo(). Error: %v", e)
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		var e error
		_, e = database.Exec(
			`INSERT OR IGNORE INTO repo (url)
			VALUES (?)`,
			value)
		if e != nil {
			errChan <- fmt.Errorf("error, when executing sql statement for addRepo(). Error: %v", e)
			return
		}
	}()

	go func() {
		wg.Wait()
		close(errChan)
	}()

	if errChanError := <-errChan; errChanError != nil {
		return "", fmt.Errorf("error, when attempting to fetch data. Error: %v", errChanError)
	}

	return "", nil
}

func isRepoValid(url string) bool {
	// Define the regular expression for a GitHub SSH URL
	re := regexp.MustCompile(`^git@github\.com:[a-zA-Z0-9._-]+\/[a-zA-Z0-9._-]+\.git$`)
	return re.MatchString(url)
}

func cloneRepo(url string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error, when fetching users home directory. Error: %v", err)
	}
	repoDir := homeDir + "/git_tool_data/repos"
	err = os.MkdirAll(repoDir, 0755)
	if err != nil {
		return fmt.Errorf("error, when creating repos directory. Error: %v", err)
	}
	_, err = os.Stat(repoDir + "/" + strings.Split(url, "/")[1])
	if os.IsNotExist(err) {
		cmd := exec.Command("git", "clone", "--bare", url)
		cmd.Dir = repoDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error, when executing clone commmand for %s. Output: %s. Error: %v", url, output, err)
		}
	} else if err != nil {
		return fmt.Errorf("error, when checking if the repo already exists. Error: %v", err)
	}
	return nil
}
