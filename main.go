package main

import (
	"embed"
	"fmt"
	"log"
	"sync"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

//go:embed schema/*
var databaseFiles embed.FS

var dataDirectory string
var reposDirectory string
var effortsDirectory string

var docStyle = lipgloss.NewStyle().
	Bold(true).
	PaddingTop(2).
	PaddingLeft(4).
	Margin(1, 2)

type model struct {
	addNewRepoTextInput             textinput.Model
	addNewEffortNameTextInput       textinput.Model
	addNewEffortBranchNameTextInput textinput.Model
	listFilterTextInput             textinput.Model
	repos                           list.Model
	efforts                         list.Model
	effortRepoVisibleSelection      []repo
	selectedEffort                  effort
	activeView                      viewOption
	loading                         bool
	loadingFinished                 chan model
	spinner                         spinner.Model
	// a filter is being created
	listFilterLive bool
	// a filter has been applied to the list
	listFilterSet bool
	cursor        int
	err           error
	validationMsg string
}

type viewOption string

const (
	activeViewAddNewRepo   viewOption = "anr"
	activeViewListRepos    viewOption = "lr"
	activeViewListEfforts  viewOption = "le"
	activeViewAddNewEffort viewOption = "ane"
	activeViewEditEffort   viewOption = "ee"
)

var deleteItemKeyBinding = key.NewBinding(
	key.WithKeys("d"),
	key.WithHelp("d", "delete"),
)

var addItemKeyBinding = key.NewBinding(
	key.WithKeys("a"),
	key.WithHelp("a", "add"),
)

var navigateToEffortsBinding = key.NewBinding(
	key.WithKeys("e"),
	key.WithHelp("e", "efforts"),
)

var navigateToReposBinding = key.NewBinding(
	key.WithKeys("r"),
	key.WithHelp("r", "repos"),
)

func initModel() (model, error) {
	var wg sync.WaitGroup
	errChan := make(chan error, 1)

	var repos []list.Item
	var efforts []list.Item

	wg.Add(1)
	go func() {
		defer wg.Done()
		var e error
		repos, e = fetchRepos()
		if e != nil {
			errChan <- fmt.Errorf("error, when fetchRepos() for initModel(). Error: %v", e)
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		var e error
		efforts, e = fetchEfforts()
		if e != nil {
			errChan <- fmt.Errorf("error, when fetchEfforts() for initModel(). Error: %v", e)
			return
		}
	}()

	go func() {
		wg.Wait()
		close(errChan)
	}()

	if errChanError := <-errChan; errChanError != nil {
		return model{}, fmt.Errorf("error, when attempting to fetch data. Error: %v", errChanError)
	}

	repoTextInput := textinput.New()
	repoTextInput.Placeholder = "git@github.com:JeremiahVaughan/git-tool.git"
	repoTextInput.Focus()
	repoTextInput.CharLimit = 256
	repoTextInput.Width = 100

	effortTextInput := textinput.New()
	effortTextInput.Placeholder = "create UI to display inventory"
	effortTextInput.CharLimit = 256
	effortTextInput.Width = 50
	effortTextInput.Focus()

	effortBranchNameTextInput := textinput.New()
	effortBranchNameTextInput.Placeholder = "Ticket ID"
	effortBranchNameTextInput.CharLimit = 32
	effortBranchNameTextInput.Width = 50

	listFilter := textinput.New()
	listFilter.Placeholder = "no active filter"
	listFilter.CharLimit = 50
	listFilter.Width = 50

	theRepos := list.New(repos, list.NewDefaultDelegate(), 0, 0) // will set width and height later
	theRepos.Title = "Repos"
	theRepos.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			addItemKeyBinding,
			deleteItemKeyBinding,
			navigateToEffortsBinding,
		}
	}

	theEfforts := list.New(efforts, list.NewDefaultDelegate(), 0, 0)
	theEfforts.Title = "Efforts"
	theEfforts.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			addItemKeyBinding,
			deleteItemKeyBinding,
			navigateToReposBinding,
		}
	}

	m := model{
		addNewRepoTextInput:             repoTextInput,
		addNewEffortNameTextInput:       effortTextInput,
		addNewEffortBranchNameTextInput: effortBranchNameTextInput,
		listFilterTextInput:             listFilter,
		repos:                           theRepos,
		activeView:                      activeViewListEfforts,
		efforts:                         theEfforts,
		err:                             nil,
		loadingFinished:                 make(chan model, 1),
	}
	m.resetSpinner()
	return m, nil
}

func main() {
	err := ProcessSchemaChanges(databaseFiles)
	if err != nil {
		log.Fatalf("error, when processing schema changes. Error: %v", err)
	}

	m, err := initModel()
	if err != nil {
		log.Fatalf("error, when initModel() for main(). Error: %v", err)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err = p.Run(); err != nil {
		log.Fatalf("error, during program run. Error: %v", err)
	}
}

type (
	errMsg error
)

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
	)
}

func (m *model) resetSpinner() {
	s := spinner.New()
	s.Spinner = spinner.Globe
	m.spinner = s
}
