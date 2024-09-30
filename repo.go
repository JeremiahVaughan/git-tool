package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/list"
)

type repo struct {
	Id          int64
	Url         string
	TrunkBranch string
	Selected    bool
	Visible     bool
}

func (r repo) Title() string {
	return strings.TrimSuffix(r.getRepoDirectoryName(), ".git")
}
func (r repo) getRepoDirectoryName() string {
	urlParts := strings.Split(r.Url, "/")
	return urlParts[len(urlParts)-1]
}
func (r repo) Description() string { return r.Url }
func (r repo) FilterValue() string { return r.Url }

func addRepo(value string) (validationMsg string, err error) {
	time.Sleep(3 * time.Second) // todo remove
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
	err := os.MkdirAll(repoDirectory, 0755)
	if err != nil {
		return fmt.Errorf("error, when creating repos directory. Error: %v", err)
	}
	_, err = os.Stat(repoDirectory + "/" + strings.Split(url, "/")[1])
	if os.IsNotExist(err) {
		cmd := exec.Command("git", "clone", "--bare", url)
		cmd.Dir = repoDirectory
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error, when executing clone commmand for %s. Output: %s. Error: %v", url, output, err)
		}
	} else if err != nil {
		return fmt.Errorf("error, when checking if the repo already exists. Error: %v", err)
	}
	return nil
}

func fetchRepos() ([]list.Item, error) {
	rows, err := database.Query(
		`SELECT id, url
		FROM repo r`,
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

	var result []repo
	for rows.Next() {
		var r repo
		err = rows.Scan(
			&r.Id,
			&r.Url,
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

	repos := make([]list.Item, len(result))
	for i, r := range result {
		repos[i] = r
	}
	if len(repos) == 0 {
		repos = []list.Item{}
	}
	return repos, nil
}

func updateRepos(allRepos []list.Item, searchString string, filteredSelectionList []repo) []list.Item {
	filteredListIndex := 0
	for i, r := range allRepos {
		theRepo := r.(repo)
		// make selections made through filtered list apply to actual list
		if len(filteredSelectionList) > filteredListIndex {
			filterRepo := filteredSelectionList[filteredListIndex]
			if filterRepo.Id == theRepo.Id {
				theRepo.Selected = filterRepo.Selected
				filteredListIndex++
			}
		}
		// set visible state based on filter value
		theRepo.Visible = searchString == "" || strings.Contains(theRepo.Title(), searchString)
		allRepos[i] = theRepo
	}
	return allRepos
}

func updateRepoVisibleSelectionList(allRepos []list.Item) []repo {
	var result []repo
	for _, r := range allRepos {
		theRepo := r.(repo)
		if theRepo.Visible {
			result = append(result, theRepo)
		}
	}
	if len(result) == 0 {
		return []repo{}
	}
	return result
}

func resetRepoSelection(repos []repo) []repo {
	for i, r := range repos {
		r.Visible = true
		r.Selected = false
		repos[i] = r
	}
	return repos
}
