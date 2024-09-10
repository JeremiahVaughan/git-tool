package main

import (
	"embed"
	"fmt"
	"log"
	"sync"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

//go:embed schema/*
var databaseFiles embed.FS

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type model struct {
	addNewRepo    textinput.Model
	addNewEffort  textinput.Model
	repos         list.Model
	efforts       list.Model
	activeView    viewOption
	err           error
	validationMsg string
}

type viewOption string

const (
	activeViewAddNewRepo   viewOption = "anr"
	activeViewListRepos    viewOption = "lr"
	activeViewListEfforts  viewOption = "le"
	activeViewAddNewEffort viewOption = "ane"
)

var deleteItemKeyBinding = key.NewBinding(
	key.WithKeys("d"),
	key.WithHelp("d", "delete"),
)

var addItemKeyBinding = key.NewBinding(
	key.WithKeys("a"),
	key.WithHelp("a", "add"),
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
	repoTextInput.Width = 50

	effortTextInput := textinput.New()
	effortTextInput.Placeholder = "create UI to display inventory"
	effortTextInput.Focus()
	effortTextInput.CharLimit = 256
	effortTextInput.Width = 50

	theRepos := list.New(repos, list.NewDefaultDelegate(), 0, 0) // will set width and height later
	theRepos.Title = "Your Repos"
	theRepos.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			addItemKeyBinding,
			deleteItemKeyBinding,
		}
	}

	theEfforts := list.New(efforts, list.NewDefaultDelegate(), 0, 0)
	theEfforts.Title = "Efforts"
	theEfforts.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			addItemKeyBinding,
			deleteItemKeyBinding,
		}
	}

	return model{
		addNewRepo:   repoTextInput,
		addNewEffort: effortTextInput,
		repos:        theRepos,
		activeView:   activeViewListEfforts,
		efforts:      theEfforts,
		err:          nil,
	}, nil
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
	return textinput.Blink
}
