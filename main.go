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
	addNewRepo textinput.Model
	repos      list.Model
	activeView viewOption
	err        error
}

type viewOption string

const (
	addNewRepo viewOption = "anr"
	listRepos  viewOption = "lr"
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

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, addItemKeyBinding) {
			// todo add new repo functionality
		} else if key.Matches(msg, deleteItemKeyBinding) {
			// todo add delete repo functionality
		}
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.repos.SetSize(msg.Width-h, msg.Height-v)
	case errMsg:
		m.err = msg
		return m, nil
	}

	// m.addNewRepo, cmd = m.addNewRepo.Update(msg)
	m.repos, cmd = m.repos.Update(msg)
	return m, cmd
}

func (m model) View() string {
	// if len(m.repos.SetShowFilter(true)) == 0 {
	// }

	return docStyle.Render(m.repos.View())
	// return fmt.Sprintf(
	// 	"Add a repo\n%s\n",
	// 	m.addNewRepo.View(),
	// )
}
