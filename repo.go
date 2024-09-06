package main

import (
	"fmt"
	"regexp"
)

func addRepo(value string) (validationMsg string, err error) {
	if value == "" {
		return "must provide a value", nil
	}
	if !isRepoValid(value) {
		return fmt.Sprintf("%s is not valid, you must provide a valid repo ssh clone url (e.g., git@github.com:JeremiahVaughan/strength-gadget-v5.git)", value), nil
	}




	// todo clone repo then update model
	_, err = database.Exec(
		`INSERT OR IGNORE INTO repo (url)
			VALUES (?)`,
		value)
	if err != nil {
		return "", fmt.Errorf("error, when executing sql statement for addRepo(). Error: %v", err)
	}
	return "", nil
}

func isRepoValid(url string) bool {
	// Define the regular expression for a GitHub SSH URL
	re := regexp.MustCompile(`^git@github\.com:[a-zA-Z0-9._-]+\/[a-zA-Z0-9._-]+\.git$`)
	return re.MatchString(url)
}
