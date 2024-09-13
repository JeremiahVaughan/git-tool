package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
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
		case activeViewListEfforts:
			if m.repos.FilterState() != list.Filtering {
				if key.Matches(msg, addItemKeyBinding) {
					m.activeView = activeViewAddNewEffort
					return m, cmd
				} else if key.Matches(msg, deleteItemKeyBinding) {
					// todo remove git tree and update model with list after item removed
				} else if key.Matches(msg, navigateToReposBinding) {
					m.activeView = activeViewListRepos
					return m, cmd
				}
				switch msg.Type {
				case tea.KeyEnter:
					m.selectedEffort = m.efforts.SelectedItem().(effort)
					m.effortRepoSelection = make([]bool, len(m.repos.Items()))
					m.activeView = activeViewEditEffort
				}
			}
		case activeViewListRepos:
			if m.repos.FilterState() != list.Filtering {
				if key.Matches(msg, addItemKeyBinding) {
					m.activeView = activeViewAddNewRepo
					return m, cmd
				} else if key.Matches(msg, deleteItemKeyBinding) {
					// todo remove git tree and update model with list after item removed
				} else if key.Matches(msg, navigateToEffortsBinding) {
					m.activeView = activeViewListEfforts
					return m, cmd
				}
			}
		case activeViewAddNewRepo:
			switch msg.Type {
			case tea.KeyEsc:
				m.activeView = activeViewListRepos
			case tea.KeyEnter:
				validationMsg, err := addRepo(m.addNewRepo.Value())
				if err != nil || validationMsg != "" {
					m.err = err
					m.validationMsg = validationMsg
					return m, cmd
				}
				m.err = nil
				m.addNewRepo.Reset()
				m.validationMsg = ""

				repos, err := fetchRepos()
				if err != nil {
					m.err = fmt.Errorf("error, when fetchRepos() for Update(). Error: %v", err)
					return m, cmd
				}
				m.repos.SetItems(repos)
				m.activeView = activeViewListRepos
			}
		case activeViewAddNewEffort:
			switch msg.Type {
			case tea.KeyEsc:
				m.activeView = activeViewListEfforts
			case tea.KeyEnter:
				validationMsg, err := addEffort(m.addNewEffort.Value())
				if err != nil || validationMsg != "" {
					m.err = err
					m.validationMsg = validationMsg
					return m, cmd
				}
				m.err = nil
				m.addNewEffort.Reset()
				m.validationMsg = ""

				efforts, err := fetchEfforts()
				if err != nil {
					m.err = fmt.Errorf("error, when fetchEfforts() for Update(). Error: %v", err)
					return m, cmd
				}
				m.efforts.SetItems(efforts)
				m.activeView = activeViewListEfforts
			}
		case activeViewEditEffort:
			switch msg.Type {
			case tea.KeyEsc:
				m.activeView = activeViewListEfforts
			case tea.KeyEnter:
				// todo save selection to DB
				m.activeView = activeViewListEfforts
			case tea.KeySpace:
				m.effortRepoSelection[m.cursor] = !m.effortRepoSelection[m.cursor]
			}
			switch msg.String() {
			case "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "j":
				if m.cursor < len(m.repos.Items())-1 {
					m.cursor++
				}
			}
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.repos.SetSize(msg.Width-h, msg.Height-v)
		m.efforts.SetSize(msg.Width-h, msg.Height-v)
	case errMsg:
		m.err = msg
		return m, nil
	}

	// if I don't do this down here the updates don't work properly, seems casting to type is causing this
	switch m.activeView {
	case activeViewListRepos:
		m.repos, cmd = m.repos.Update(msg)
	case activeViewAddNewRepo:
		m.addNewRepo, cmd = m.addNewRepo.Update(msg)
	case activeViewListEfforts:
		m.efforts, cmd = m.efforts.Update(msg)
	case activeViewAddNewEffort:
		m.addNewEffort, cmd = m.addNewEffort.Update(msg)
	}
	return m, cmd
}
