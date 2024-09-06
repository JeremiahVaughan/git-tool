package main

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}

		switch m.activeView {
		case activeViewListRepos:
			if key.Matches(msg, addItemKeyBinding) {
				m.activeView = activeViewAddNewRepo
			} else if key.Matches(msg, deleteItemKeyBinding) {
				// todo remove git tree and update model with list after item removed
			}
			m.repos, cmd = m.repos.Update(msg)
		case activeViewAddNewRepo:
			switch msg.Type {
			case tea.KeyEsc:
				m.activeView = activeViewListRepos
			case tea.KeyEnter:
				validationMsg, err := addRepo(m.addNewRepo.Value())
				if err != nil || validationMsg != "" {
					m.err = err
					m.validationMsg = validationMsg
				} else {
					m.err = nil
					m.addNewRepo.Reset()
					m.validationMsg = ""
					m.activeView = activeViewListRepos
				}
			}
			m.addNewRepo, cmd = m.addNewRepo.Update(msg)
		}
		return m, cmd
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.repos.SetSize(msg.Width-h, msg.Height-v)
	case errMsg:
		m.err = msg
		return m, nil
	}

	return m, cmd
}
