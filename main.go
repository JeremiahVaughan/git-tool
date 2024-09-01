package main

import (
	"embed"
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

//go:embed schema/*
var databaseFiles embed.FS

type model struct {
	addNewRepo textinput.Model
	err        error
}

func initModel() model {
	ti := textinput.New()
	ti.Placeholder = "git@github.com:JeremiahVaughan/git-tool.git"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50

	return model{
		addNewRepo: ti,
		err:        nil,
	}
}

func main() {
	p := tea.NewProgram(initModel())
	if _, err := p.Run(); err != nil {
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
		switch msg.Type {
		case tea.KeyEnter, tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.addNewRepo, cmd = m.addNewRepo.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return fmt.Sprintf(
		"Add a repo\n%s\n",
		m.addNewRepo.View(),
	)
}
