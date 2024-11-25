package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/list"
)

type effort struct {
	Id         int64
	Name       string
	BranchName string
	Desc       string
	Repos      []repo
}

func (e effort) Title() string {
	return e.Name
}
func (e effort) Description() string { return e.Desc }
func (e effort) FilterValue() string { return e.Desc }

func addEffort(effortName, branchName string) (string, error) {
	effortName = strings.TrimSpace(effortName)
	if effortName == "" {
		return "must provide a name", nil
	}

	description := effortName
	effortName = strings.ToLower(effortName)
	name := strings.ReplaceAll(effortName, " ", "_")

	branchName = strings.TrimSpace(branchName)
	if branchName == "" {
		branchName = name
	}

	var wg sync.WaitGroup
	errChan := make(chan error, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()
		var e error
		effortDir, e := getEffortDir(name)
		if e != nil {
			errChan <- fmt.Errorf("error, when getting effort directory. Error: %v", e)
			return
		}
		e = os.MkdirAll(effortDir, 0755)
		if e != nil {
			errChan <- fmt.Errorf("error, when creating effort directory. Error: %v", e)
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		var e error
		_, e = database.Exec(
			`INSERT OR IGNORE INTO effort (name, branch_name, description)
			VALUES (?, ?, ?)`,
			name,
			branchName,
			description,
		)
		if e != nil {
			errChan <- fmt.Errorf("error, when executing sql statement to add effort. Error: %v", e)
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
		`SELECT id, name, branch_name, description
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
			&r.BranchName,
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

func applyRepoSelectionForEffort(theEffort effort, repos []list.Item) (string, error) {
	var selected []repo
	var notSelected []repo
	for _, r := range repos {
		theRepo := r.(repo)
		if theRepo.Selected {
			selected = append(selected, theRepo)
		} else {
			notSelected = append(notSelected, theRepo)
		}
	}
	if len(selected) == 0 {
		return "must select at least one repo", nil
	}

	var wg sync.WaitGroup
	errChan := make(chan error, 1)

	err := os.MkdirAll(effortsDirectory, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("error, when creating effort directory for applyRepoSelectionForEffort(). Error: %v", err)
	}

	for _, theRepo := range selected {
		wg.Add(1)
		go func(r repo) {
			defer wg.Done()
			e := createWorktree(theEffort, r)
			if e != nil {
				errChan <- fmt.Errorf("error, when createWorktree() for applyRepoSelectionForEffort() of key: %s. Error: %v", r.Title(), e)
				return
			}
		}(theRepo)
	}

	for _, theRepo := range notSelected {
		wg.Add(1)
		go func(r repo) {
			defer wg.Done()
			e := deleteWorktree(theEffort, r)
			if e != nil {
				errChan <- fmt.Errorf("error, when deleteWorktree() for applyRepoSelectionForEffort() of key: %s. Error: %v", r.Title(), e)
				return
			}
		}(theRepo)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	if errChanError := <-errChan; errChanError != nil {
		return "", fmt.Errorf("error, when attempting perform async actions. Error: %v", errChanError)
	}

	err = persistRepoSelection(theEffort.Id, selected)
	if err != nil {
		return "", fmt.Errorf("error, when persistRepoSelection() for applyRepoSelectionForEffort(). Error: %v", err)
	}
	return "", nil
}

func createWorktree(theEffort effort, r repo) error {
	worktreeDir := getWorktreeDir(theEffort, r)
	alreadyExists, err := checkDirectoryExists(worktreeDir)
	if err != nil {
		return fmt.Errorf("error, when checkDirectoryExists() for createWorktree(). Error: %v", err)
	}
	if !alreadyExists {
		command := "git"
		commandDir := reposDirectory + r.getRepoDirectoryName()
		commmandParts := []string{"worktree", "add", worktreeDir}
		branchAlreadyExists, err := doesBranchExist(theEffort.BranchName, commandDir)
		if err != nil {
			return fmt.Errorf("error, when doesBranchExist() for createWorktree(). Error: %v", err)
		}
		if !branchAlreadyExists {
			commmandParts = append(commmandParts, "-b")
		}
		commmandParts = append(commmandParts, theEffort.BranchName)
		cmd := exec.Command(command, commmandParts...)
		cmd.Dir = commandDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			commandString := command + " " + strings.Join(commmandParts, " ")
			return fmt.Errorf("error, when creating worktree with command: %s at directory: %s. Output: %s, Error: %v", commandString, cmd.Dir, output, err)
		}
	}
	err = ensureRemoteBranchesExists(worktreeDir, theEffort.BranchName)
	if err != nil {
		return fmt.Errorf("error, when ensureRemoteBranchExists() for createWorktree(). Error: %v", err)
	}
	return nil
}

func ensureRemoteBranchesExists(worktreeDir string, branchName string) (err error) {
	commandDir := worktreeDir
	command := "git"

	defer func() {
		commandParts := []string{"switch", branchName}
		cmd := exec.Command(command, commandParts...)
		cmd.Dir = commandDir
		output, cleanupErr := cmd.CombinedOutput()
		if cleanupErr != nil {
			commandString := command + " " + strings.Join(commandParts, " ")
			err = fmt.Errorf(
				"error, when executing command for ensureRemoteBranchExists(). Command: %s at directory: %s. Output: %s, Error: %v, Cleanup Error: %v",
				commandString,
				commandDir,
				output,
				err,
				cleanupErr,
			)
		}
	}()

	commandParts := []string{"push", "-u", "origin", branchName}
	cmd := exec.Command(command, commandParts...)
	cmd.Dir = commandDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		commandString := command + " " + strings.Join(commandParts, " ")
		return fmt.Errorf(
			"error, when executing command for ensureRemoteBranchExists(). Command: %s at directory: %s. Output: %s, Error: %v",
			commandString,
			commandDir,
			output,
			err,
		)
	}

	commandParts = []string{"switch", "master"}
	cmd = exec.Command(command, commandParts...)
	cmd.Dir = commandDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		commandString := command + " " + strings.Join(commandParts, " ")
		return fmt.Errorf(
			"error, when executing command for ensureRemoteBranchExists(). Command: %s at directory: %s. Output: %s, Error: %v",
			commandString,
			commandDir,
			output,
			err,
		)
	}

	commandParts = []string{"pull", "origin", "master"}
	cmd = exec.Command(command, commandParts...)
	cmd.Dir = commandDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		commandString := command + " " + strings.Join(commandParts, " ")
		return fmt.Errorf(
			"error, when executing command for ensureRemoteBranchExists(). Command: %s at directory: %s. Output: %s, Error: %v",
			commandString,
			commandDir,
			output,
			err,
		)
	}

	commandParts = []string{"push", "-u", "origin", "master"}
	cmd = exec.Command(command, commandParts...)
	cmd.Dir = commandDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		commandString := command + " " + strings.Join(commandParts, " ")
		return fmt.Errorf(
			"error, when executing command for ensureRemoteBranchExists(). Command: %s at directory: %s. Output: %s, Error: %v",
			commandString,
			commandDir,
			output,
			err,
		)
	}
	return nil
}

func verifySafeDeletionOfRemoteBranch(worktreeDir string, theEffort effort, r repo) (err error) {
	commandDir := worktreeDir
	command := "git"

	defer func() {
		commandParts := []string{"switch", theEffort.BranchName}
		cmd := exec.Command(command, commandParts...)
		cmd.Dir = commandDir
		output, cleanupErr := cmd.CombinedOutput()
		if cleanupErr != nil {
			commandString := command + " " + strings.Join(commandParts, " ")
			err = fmt.Errorf(
				"error, when verifying if its safe to delete worktree with command: %s at directory: %s. Output: %s, Error: %v, Cleanup Error: %v",
				commandString,
				commandDir,
				output,
				err,
				cleanupErr,
			)
		}
	}()

	commandParts := []string{"status"}
	cmd := exec.Command(command, commandParts...)
	cmd.Dir = commandDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		commandString := command + " " + strings.Join(commandParts, " ")
		return fmt.Errorf("error, when verifying if its safe to delete worktree with command: %s at directory: %s. Output: %s, Error: %v", commandString, commandDir, output, err)
	}
	if !strings.Contains(string(output), "working tree clean") {
		return fmt.Errorf("unsafe delete operation, please stash or commit your changes. Effort: %s. Repo: %s", theEffort.Name, r.Title())
	}

	// pulling and pushing any existing changes on both the working branch and master to get local in sync with remote
	// pulling first since remote should always be the source of truth
	commandParts = []string{"pull"}
	cmd = exec.Command(command, commandParts...)
	cmd.Dir = commandDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		commandString := command + " " + strings.Join(commandParts, " ")
		return fmt.Errorf("error, when verifying if its safe to delete worktree with command: %s at directory: %s. Output: %s, Error: %v", commandString, commandDir, output, err)
	}
	commandParts = []string{"push"}
	cmd = exec.Command(command, commandParts...)
	cmd.Dir = commandDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		commandString := command + " " + strings.Join(commandParts, " ")
		return fmt.Errorf("error, when verifying if its safe to delete worktree with command: %s at directory: %s. Output: %s, Error: %v", commandString, commandDir, output, err)
	}
	commandParts = []string{"switch", "master"}
	cmd = exec.Command(command, commandParts...)
	cmd.Dir = commandDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		commandString := command + " " + strings.Join(commandParts, " ")
		return fmt.Errorf("error, when verifying if its safe to delete worktree with command: %s at directory: %s. Output: %s, Error: %v", commandString, commandDir, output, err)
	}
	commandParts = []string{"pull"}
	cmd = exec.Command(command, commandParts...)
	cmd.Dir = commandDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		commandString := command + " " + strings.Join(commandParts, " ")
		return fmt.Errorf("error, when verifying if its safe to delete worktree with command: %s at directory: %s. Output: %s, Error: %v", commandString, commandDir, output, err)
	}
	commandParts = []string{"push"}
	cmd = exec.Command(command, commandParts...)
	cmd.Dir = commandDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		commandString := command + " " + strings.Join(commandParts, " ")
		return fmt.Errorf("error, when verifying if its safe to delete worktree with command: %s at directory: %s. Output: %s, Error: %v", commandString, commandDir, output, err)
	}

	commandParts = []string{"branch", "--no-merged"}
	cmd = exec.Command(command, commandParts...)
	cmd.Dir = commandDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		commandString := command + " " + strings.Join(commandParts, " ")
		return fmt.Errorf("error, when verifying if its safe to delete worktree with command: %s at directory: %s. Output: %s, Error: %v", commandString, commandDir, output, err)
	}
	if strings.Contains(string(output), theEffort.BranchName) {
		err = fmt.Errorf(
			"cannot delete the remote branch until it has been merged into master for effort: %s. repo: %s. Branch: %s",
			theEffort.Name,
			r.Title(),
			theEffort.BranchName,
		)
	}

	return err
}

func deleteWorktree(theEffort effort, r repo) error {
	worktreeDir := getWorktreeDir(theEffort, r)
	command := "git"
	commandDir := reposDirectory + r.getRepoDirectoryName()
	exists, err := checkDirectoryExists(worktreeDir)
	if err != nil {
		return fmt.Errorf("error, when checkDirectoryExists() for deteletWorktree(). Error: %v", err)
	}
	if exists {
		// we delete the branches first because we need the worktree directory to perform validation
		branchExists, err := doesBranchExist(theEffort.BranchName, commandDir)
		if err != nil {
			return fmt.Errorf("error, when doesBranchExist() for deteletWorktree. Error: %v", err)
		}
		if branchExists {
			err := ensureWorktreeIsOnCorrectBranch(worktreeDir, theEffort.BranchName)
			if err != nil {
				return fmt.Errorf("error, when ensureWorktreeIsOnCorrectBranch() for deteletWorktree(). Error: %v", err)
			}

			remoteBranchExists, err := doesRemoteBranchExist(theEffort.BranchName, commandDir)
			if err != nil {
				return fmt.Errorf("error, when doesBranchExist() for deteletWorktree. Error: %v", err)
			}
			if remoteBranchExists {
				err = verifySafeDeletionOfRemoteBranch(worktreeDir, theEffort, r)
				if err != nil {
					return fmt.Errorf("error, when verifySafeDeleteOfRemoteBranch() for deleteWorktree(). Error: %v", err)
				}
				deleteBranchRemoteCmdParts := []string{"push", "origin", "--delete", theEffort.BranchName}
				deleteBranchRemoteCmd := exec.Command(command, deleteBranchRemoteCmdParts...)
				deleteBranchRemoteCmd.Dir = commandDir
				output, err := deleteBranchRemoteCmd.CombinedOutput()
				if err != nil {
					commandString := command + " " + strings.Join(deleteBranchRemoteCmdParts, " ")
					return fmt.Errorf("error, when deleting remote branch: %s at directory: %s. Output: %s, Error: %v", commandString, commandDir, output, err)
				}
			}

			// cannot delete a branch while we are on that branch, so switching to master
			commandParts := []string{"switch", "master"}
			cmd := exec.Command(command, commandParts...)
			cmd.Dir = worktreeDir
			output, err := cmd.CombinedOutput()
			if err != nil {
				commandString := command + " " + strings.Join(commandParts, " ")
				return fmt.Errorf(
					"error, when verifying if its safe to delete worktree with command: %s at directory: %s. Output: %s, Error: %v",
					commandString,
					commandDir,
					output,
					err,
				)
			}

			deleteBranchCmdParts := []string{"branch", "-d", theEffort.BranchName}
			deleteBranchCmd := exec.Command(command, deleteBranchCmdParts...)
			deleteBranchCmd.Dir = commandDir
			output, err = deleteBranchCmd.CombinedOutput()
			if err != nil {
				commandString := command + " " + strings.Join(deleteBranchCmdParts, " ")
				return fmt.Errorf("error, when deleting worktree with command: %s at directory: %s. Output: %s, Error: %v", commandString, commandDir, output, err)
			}
		}

		commmandParts := []string{"worktree", "remove", worktreeDir}
		cmd := exec.Command(command, commmandParts...)
		cmd.Dir = commandDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			commandString := command + " " + strings.Join(commmandParts, " ")
			return fmt.Errorf("error, when deleting worktree with command: %s at directory: %s. Output: %s, Error: %v", commandString, commandDir, output, err)
		}
	}

	return nil
}

func ensureWorktreeIsOnCorrectBranch(worktreeDir string, branchName string) error {
	command := "git"
	commandDir := worktreeDir
	commmandParts := []string{"switch", branchName}
	cmd := exec.Command(command, commmandParts...)
	cmd.Dir = commandDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		commandString := command + " " + strings.Join(commmandParts, " ")
		return fmt.Errorf("error, when ensuring worktree is on the correct branch with command: %s at directory: %s. Output: %s, Error: %v", commandString, commandDir, output, err)
	}
	return nil
}

func persistRepoSelection(effortId int64, repos []repo) error {
	err := deleteAnyNoLongerSelected(effortId, repos)
	if err != nil {
		return fmt.Errorf("error, when deleteAnyNoLongerSelected() for persistRepoSelection(). Error: %v", err)
	}

	till := len(repos) * 2
	args := make([]any, till)
	j := 0
	for i := 0; i < till; i += 2 {
		args[i] = effortId
		args[i+1] = repos[j].Id
		j++
	}
	insertStatement := generateRepoSelectionInsertStatement(repos)
	_, err = database.Exec(
		insertStatement,
		args...,
	)
	if err != nil {
		return fmt.Errorf("error, when executing insert records statement for persistRepoSelection(). Error: %v", err)
	}
	return nil
}

func generateRepoSelectionInsertStatement(repos []repo) string {
	columns := []string{"effort_id", "repo_id"}
	result := `INSERT OR IGNORE INTO effort_repo (%s)
		VALUES %s`
	placeholders := make([]string, len(repos))
	for i := range repos {
		innerPlaceholders := make([]string, len(columns))
		for j := range columns {
			innerPlaceholders[j] = "?"
		}
		placeholders[i] = "(" + strings.Join(innerPlaceholders, ", ") + ")"
	}
	result = fmt.Sprintf(
		result,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)
	return result
}

func deleteAnyNoLongerSelected(effortId int64, repos []repo) error {
	placeholders := make([]string, len(repos))
	args := make([]any, len(repos)+1)
	args[0] = effortId
	for i, r := range repos {
		placeholders[i] = "?"
		args[i+1] = r.Id
	}
	deleteStatement := fmt.Sprintf(
		`DELETE FROM effort_repo
		WHERE effort_id = ?
		AND repo_id NOT IN (%s)`,
		strings.Join(placeholders, ", "),
	)
	_, err := database.Exec(
		deleteStatement,
		args...,
	)
	if err != nil {
		return fmt.Errorf("error, when executing sql statement. Statement: %s. Error: %v", deleteStatement, err)
	}
	return nil
}

func fetchSelectedReposForEffort(effortId int64) (map[int64]bool, error) {
	rows, err := database.Query(
		`SELECT repo_id
		FROM effort_repo
		WHERE effort_id = ?`,
		effortId,
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

	// selectedReposMap key is repo id and value doesn't matter
	selectedReposMap := make(map[int64]bool)
	for rows.Next() {
		var id int64
		err = rows.Scan(
			&id,
		)
		if err != nil {
			return nil, fmt.Errorf("error, when scanning database rows. Error: %v", err)
		}
		selectedReposMap[id] = true
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error, when iterating through database rows. Error: %v", err)
	}
	return selectedReposMap, nil
}

func fetchEffortRepoChoices(effortId int64, allRepos list.Model) ([]list.Item, error) {
	selectedReposMap, err := fetchSelectedReposForEffort(effortId)
	if err != nil {
		return nil, fmt.Errorf("error, when fetchSelectedReposForEffort() for fetchEffortRepoChoices(). Error: %v", err)
	}
	result := make([]list.Item, len(allRepos.Items()))
	for i, r := range allRepos.Items() {
		theRepo := r.(repo)
		_, ok := selectedReposMap[theRepo.Id]
		if ok {
			theRepo.Selected = true
		} else {
			theRepo.Selected = false
		}
		theRepo.Visible = true
		result[i] = theRepo
	}
	return result, nil

}

func doesBranchExist(branchName string, commandDir string) (bool, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", branchName)
	cmd.Dir = commandDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputString := string(output)
		if strings.Contains(outputString, "Needed a single revision") {
			return false, nil // Branch does not exist
		}
		// If there was another error, return it
		return false, fmt.Errorf("error, when running command. Output %s. Error: %v", output, err)
	}
	// If no error, the branch exists
	return true, nil
}

func doesRemoteBranchExist(branchName string, commandDir string) (bool, error) {
	cmd := exec.Command("git", "ls-remote", "--heads", "origin", branchName)
	cmd.Dir = commandDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If there was another error, return it
		return false, fmt.Errorf("error, when running command. Output %s. Error: %v", output, err)
	}
	outputString := string(output)
	if !strings.Contains(outputString, branchName) {
		return false, nil // remote branch does not exist
	}
	// If no error, the branch exists
	return true, nil
}

func checkDirectoryExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("error, when checking if directory exists for checkDirectoryExists(). Error: %v", err)
	}
	return info.IsDir(), nil
}

func deleteEffort(theEffort effort) error {
	repoIds, err := fetchSelectedReposForEffort(theEffort.Id)
	if err != nil {
		return fmt.Errorf("error, when fetchSelectedReposForEffort() for deleteEffort(). Error: %v", err)
	}
	effortRepos, err := fetchReposForIds(repoIds)
	if err != nil {
		return fmt.Errorf("error, when fetchReposForIds() for deleteEffort(). Error: %v", err)
	}
	if len(effortRepos) != 0 {
		for _, r := range effortRepos {
			err = deleteWorktree(theEffort, r)
			if err != nil {
				return fmt.Errorf("error, when deleteWorktree() for deleteEffort(). Error: %v", err)
			}
		}
		theStatement := `DELETE FROM effort_repo
                    WHERE effort_id = ?`
		_, err = database.Exec(theStatement, theEffort.Id)
		if err != nil {
			return fmt.Errorf("error, when deleting from effort_repo table for deleteEffort(). Error: %v", err)
		}
	}
	effortDir, err := getEffortDir(theEffort.Name)
	if err != nil {
		return fmt.Errorf("error, when getEffortDir() for deleteEffort(). Error: %v", err)
	}
	err = os.Remove(effortDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error, when os.Remove() for deleteEffort(). Error: %v", err)
	}
	theStatement := `DELETE FROM effort
                    WHERE id = ?`
	_, err = database.Exec(theStatement, theEffort.Id)
	if err != nil {
		return fmt.Errorf("error, when deleting from effort table for deleteEffort(). Error: %v", err)
	}
	return nil
}

func getWorktreeDir(theEffort effort, r repo) string {
	return fmt.Sprintf("%s%s/%s", effortsDirectory, theEffort.Name, r.Title())
}

func fetchReposForIds(repoIds map[int64]bool) ([]repo, error) {
	placeholders := make([]string, len(repoIds))
	args := make([]any, len(repoIds))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	i := 0
	for k, _ := range repoIds {
		args[i] = k
		i++
	}
	theStatement := `SELECT url
                    FROM repo
                    WHERE id IN (%s)`
	theStatement = fmt.Sprintf(theStatement, strings.Join(placeholders, ","))

	rows, err := database.Query(
		theStatement,
		args...,
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
	return result, nil
}

func getEffortDir(name string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error, when fetching users home directory. Error: %v", err)
	}
	return homeDir + "/git_tool_data/efforts/" + name, nil
}
