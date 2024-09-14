package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/list"
)

type effort struct {
	Id    int64
	Name  string
	Desc  string
	Repos []repo
}

func (e effort) Title() string {
	return e.Name
}
func (e effort) Description() string { return e.Desc }
func (e effort) FilterValue() string { return e.Desc }

func addEffort(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "must provide a value", nil
	}
	description := value
	value = strings.ToLower(value)
	name := strings.ReplaceAll(value, " ", "_")

	var wg sync.WaitGroup
	errChan := make(chan error, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()
		var e error
		homeDir, err := os.UserHomeDir()
		if err != nil {
			errChan <- fmt.Errorf("error, when fetching users home directory. Error: %v", e)
			return
		}
		repoDir := homeDir + "/git_tool_data/efforts/" + name
		err = os.MkdirAll(repoDir, 0755)
		if err != nil {
			errChan <- fmt.Errorf("error, when creating effort directory. Error: %v", e)
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		var e error
		_, e = database.Exec(
			`INSERT OR IGNORE INTO effort (name, description)
			VALUES (?, ?)`,
			name,
			description,
		)
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

func fetchEfforts() ([]list.Item, error) {
	rows, err := database.Query(
		`SELECT id, name, description
		FROM effort e`,
	)

	defer func(rows *sql.Rows) {
		if rows != nil {
			closeRowsError := rows.Close()
			if closeRowsError != nil {
				// no choice but to log the error since defer doesn't let us return errors
				// defer is needed though because it ensures a cleanup attempt is made even if we should return early due to an error
				log.Printf("error, when attempting to close database rows: %v", closeRowsError)
			}
		}
	}(rows)

	if err != nil {
		return nil, fmt.Errorf("error, when attempting to retrieve records. Error: %v", err)
	}

	var result []effort
	for rows.Next() {
		var r effort
		err = rows.Scan(
			&r.Id,
			&r.Name,
			&r.Desc,
		)
		if err != nil {
			return nil, fmt.Errorf("error, when scanning database rows. Error: %v", err)
		}
		result = append(result, r)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error, when iterating through database rows. Error: %v", err)
	}

	efforts := make([]list.Item, len(result))
	for i, r := range result {
		efforts[i] = r
	}
	if len(efforts) == 0 {
		efforts = []list.Item{}
	}

	return efforts, nil
}

func applyRepoSelectionForEffort(repos []list.Item) (string, error) {
	var selected []repo
	for _, r := range repos {
		theRepo := r.(repo)
		if theRepo.Selected {
			selected = append(selected, theRepo)
		}
	}
	if len(selected) == 0 {
		return "must select at least one repo", nil
	}

	var wg sync.WaitGroup
	errChan := make(chan error, 1)

	for _, theRepo := range repos {
		wg.Add(1)
		go func(r repo) {
			defer wg.Done()
			e := createWorktree(r)
			if e != nil {
				errChan <- fmt.Errorf("error, when createWorktree() for applyRepoSelectionForEffort() of key: %s. Error: %v", r.Title(), e)
				return
			}
		}(theRepo.(repo))
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		var e error
		e = persistRepoSelection(selected)
		if e != nil {
			errChan <- fmt.Errorf("error, when persistRepoSelection() for applyRepoSelectionForEffort(). Error: %v", e)
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

func createWorktree(r repo) error {
	// todo implement
}

func persistRepoSelection(r []repo) error {
	// todo implement
}
