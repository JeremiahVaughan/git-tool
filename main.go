package main

import (
	"embed"
	"fmt"
	"log"

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
	repos         list.Model
	efforts       list.Model
	activeView    viewOption
	err           error
	validationMsg string
}

type viewOption string

const (
	activeViewAddNewRepo  viewOption = "anr"
	activeViewListRepos   viewOption = "lr"
	activeViewListEfforts viewOption = "le"
)

var deleteItemKeyBinding = key.NewBinding(
	key.WithKeys("d"),
	key.WithHelp("d", "delete repo"),
)

var addItemKeyBinding = key.NewBinding(
	key.WithKeys("a"),
	key.WithHelp("a", "add repo"),
)

func initModel() (model, error) {
	ti := textinput.New()
	ti.Placeholder = "git@github.com:JeremiahVaughan/git-tool.git"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50

	repos, err := fetchRepos()
	if err != nil {
		return model{}, fmt.Errorf("error, when fetchRepos() for initModel(). Error: %v", err)
	}

	theList := list.New(repos, list.NewDefaultDelegate(), 0, 0) // will set width and height later
	theList.Title = "Your Repos"
	theList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			addItemKeyBinding,
			deleteItemKeyBinding,
		}
	}

	return model{
		addNewRepo: ti,
		repos:      theList,
		activeView: activeViewListRepos,
		err:        nil,
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
