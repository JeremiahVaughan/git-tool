package main

import (
	"database/sql"
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
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
	if value == "" {
		return "must provide a value", nil
	}
	if !isRepoValid(value) {
		return fmt.Sprintf("%s is not valid, you must provide a valid repo ssh clone url (e.g., git@github.com:JeremiahVaughan/strength-gadget-v5.git)", value), nil
	}

	err = cloneRepo(value)
	if err != nil {
		return "", fmt.Errorf("error, when cloneRepo() for addRepo(). Error: %v", err)
	}

	_, err = database.Exec(
		`INSERT INTO repo (url)
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

func cloneRepo(url string) error {
	err := os.MkdirAll(reposDirectory, 0755)
	if err != nil {
		return fmt.Errorf("error, when creating repos directory. Error: %v", err)
	}
	_, err = os.Stat(getRepoDir(url))
	if os.IsNotExist(err) {
		cmd := exec.Command("git", "clone", "--bare", url)
		cmd.Dir = reposDirectory
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error, when executing clone commmand for %s. Output: %s. Error: %v", url, output, err)
		}
	}
	return nil
}

func getRepoDir(url string) string {
	return reposDirectory + "/" + strings.Split(url, "/")[1]
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

func deleteRepo(theRepo repo) error {
	err := isSafeToDeleteRepo(theRepo)
	if err != nil {
		return fmt.Errorf("error, when isSafeToDeleteRepo() for deleteRepo(). Error: %v", err)
	}

	cmd := exec.Command("rm", "-rf", getRepoDir(theRepo.Url))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error, when attempting to delete repo files. Output: %s. Error: %v", output, err)
	}

	_, err = database.Exec(
		`DELETE FROM repo
        WHERE id = ?`,
		theRepo.Id,
	)
	if err != nil {
		return fmt.Errorf("error, when attempting to delete repo %s from database. Error: %v", theRepo.Title(), err)
	}
	return nil
}

func isSafeToDeleteRepo(theRepo repo) error {
	rows, err := database.Query(
		`SELECT e.name
    FROM effort_repo er
    JOIN effort e ON er.effort_id = e.id
    WHERE repo_id = ?`,
		theRepo.Id,
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
		return fmt.Errorf("error, when attempting to retrieve records. Error: %v", err)
	}

	var result []string
	for rows.Next() {
		var r string
		err = rows.Scan(
			&r,
		)
		if err != nil {
			return fmt.Errorf("error, when scanning database rows. Error: %v", err)
		}
		result = append(result, r)
	}

	err = rows.Err()
	if err != nil {
		return fmt.Errorf("error, when iterating through database rows. Error: %v", err)
	}

	if len(result) != 0 {
		return fmt.Errorf(
			"unsafe delete operation, the %s repo is still being used by these efforts: %s",
			theRepo.Url,
			strings.Join(result, ", "),
		)
	}
	return nil
}
