package main

import (
	"embed"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

//go:embed schema/*
var databaseFiles embed.FS

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type repo struct {
	url         string
	trunkBranch string
}

func (r repo) Title() string {
	urlParts := strings.Split(r.url, "/")
	repoName := urlParts[len(urlParts)-1]
	return strings.TrimSuffix(repoName, ".git")
}
func (r repo) Description() string { return r.url }
func (r repo) FilterValue() string { return r.url }

type model struct {
	addNewRepo    textinput.Model
	repos         list.Model
	activeView    viewOption
	err           error
	validationMsg string
}

type viewOption string

const (
	activeViewAddNewRepo viewOption = "anr"
	activeViewListRepos  viewOption = "lr"
)

var deleteItemKeyBinding = key.NewBinding(
	key.WithKeys("d"),
	key.WithHelp("d", "delete repo"),
)

var addItemKeyBinding = key.NewBinding(
	key.WithKeys("a"),
	key.WithHelp("a", "add repo"),
)

func initModel() model {
	ti := textinput.New()
	ti.Placeholder = "git@github.com:JeremiahVaughan/git-tool.git"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50

	// test code
	repos := []list.Item{
		repo{
			url:         "git@github.com:JeremiahVaughan/git-tool.git",
			trunkBranch: "master",
		},
		repo{
			url:         "git@github.com:JeremiahVaughan/strength-gadget-v5.git",
			trunkBranch: "trunk",
		},
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
		activeView: activeViewListRepos,
		repos:      theList,
		err:        nil,
	}
}

func main() {
	err := ProcessSchemaChanges(databaseFiles)
	if err != nil {
		log.Fatalf("error, when processing schema changes. Error: %v", err)
	}

	p := tea.NewProgram(initModel(), tea.WithAltScreen())
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
