package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// ignore key presses if loading
		if !m.loading {
			// reset any errors or validation messages on key press if not loading
			m.err = nil
			m.validationMsg = ""

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
						if len(m.repos.Items()) == 0 {
							m.activeView = activeViewAddNewRepo
							return m, cmd
						}
						m.selectedEffort = m.efforts.SelectedItem().(effort)
						theRepoItems, err := fetchSelectedReposForEffort(m.selectedEffort.Id, m.repos)
						if err != nil {
							m.err = fmt.Errorf("error, when fetchSelectedReposForEffort() for Update(). Error: %v", err)
							return m, cmd
						}

						m.repos.SetItems(theRepoItems)
						m.effortRepoVisibleSelection = updateRepoVisibleSelectionList(m.repos.Items())
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
					if !m.loading {
						m.loading = true
						m.addNewRepoTextInput.Blur()
						go func() {
							validationMsg, err := addRepo(m.addNewRepoTextInput.Value())
							if err != nil || validationMsg != "" {
								m.err = err
								m.validationMsg = validationMsg
							} else {
								m.err = nil
								m.addNewRepoTextInput.Reset()
								m.validationMsg = ""
								repos, err := fetchRepos()
								if err != nil {
									m.err = fmt.Errorf("error, when fetchRepos() for Update(). Error: %v", err)
									return
								}
								m.repos.SetItems(repos)
								m.activeView = activeViewListRepos
							}
							m.loadingFinished <- m
						}()
						return m, m.spinner.Tick
					}
				}
			case activeViewAddNewEffort:
				switch msg.Type {
				case tea.KeyEsc:
					m.activeView = activeViewListEfforts
				case tea.KeyEnter:
					validationMsg, err := addEffort(
						m.addNewEffortNameTextInput.Value(),
						m.addNewEffortBranchNameTextInput.Value(),
					)
					if err != nil || validationMsg != "" {
						m.err = err
						m.validationMsg = validationMsg
						return m, cmd
					} else {
						m.err = nil
						m.addNewEffortNameTextInput.Reset()
						m.addNewEffortBranchNameTextInput.Reset()
						m.validationMsg = ""
					}

					efforts, err := fetchEfforts()
					if err != nil {
						m.err = fmt.Errorf("error, when fetchEfforts() for Update(). Error: %v", err)
						return m, cmd
					}
					m.efforts.SetItems(efforts)
					m.activeView = activeViewListEfforts
				case tea.KeyTab:
					if m.addNewEffortNameTextInput.Focused() {
						m.addNewEffortBranchNameTextInput.Focus()
						m.addNewEffortNameTextInput.Blur()
					} else {
						m.addNewEffortNameTextInput.Focus()
						m.addNewEffortBranchNameTextInput.Blur()
					}
				}
			case activeViewEditEffort:
				if m.listFilterLive {
					switch msg.Type {
					case tea.KeyEsc:
						m.listFilterLive = false
						m.listFilterSet = false
						m.listFilterTextInput.Reset()
					case tea.KeyEnter:
						m.listFilterSet = true
						m.listFilterLive = false
						m.listFilterTextInput.Blur()
					default:
						m.cursor = 0
						m.listFilterTextInput, cmd = m.listFilterTextInput.Update(msg)
						theRepos := updateRepos(
							m.repos.Items(),
							m.listFilterTextInput.Value(),
							m.effortRepoVisibleSelection,
						)
						m.repos.SetItems(theRepos)
						m.effortRepoVisibleSelection = updateRepoVisibleSelectionList(m.repos.Items())
					}
				} else {
					switch msg.Type {
					case tea.KeyEsc:
						m.activeView = activeViewListEfforts
					case tea.KeyEnter:
						validationMsg, err := applyRepoSelectionForEffort(m.selectedEffort, m.repos.Items())
						if err != nil || validationMsg != "" {
							m.err = err
							m.validationMsg = validationMsg
							return m, cmd
						}
						m.effortRepoVisibleSelection = resetRepoSelection(m.effortRepoVisibleSelection)
						m.activeView = activeViewListEfforts
					case tea.KeySpace:
						m.effortRepoVisibleSelection[m.cursor].Selected = !m.effortRepoVisibleSelection[m.cursor].Selected
						theRepos := updateRepos(
							m.repos.Items(),
							m.listFilterTextInput.Value(),
							m.effortRepoVisibleSelection,
						)
						m.repos.SetItems(theRepos)
						m.effortRepoVisibleSelection = updateRepoVisibleSelectionList(m.repos.Items())

					}
					switch msg.String() {
					case "k":
						if m.cursor > 0 {
							m.cursor--
						}
					case "j":
						if m.cursor < len(m.effortRepoVisibleSelection)-1 {
							m.cursor++
						}
					case "/":
						m.listFilterLive = true
						m.listFilterTextInput.Reset()
						m.listFilterTextInput.Focus()
						theRepos := updateRepos(
							m.repos.Items(),
							m.listFilterTextInput.Value(),
							m.effortRepoVisibleSelection,
						)
						m.repos.SetItems(theRepos)
						m.effortRepoVisibleSelection = updateRepoVisibleSelectionList(m.repos.Items())
						return m, cmd
					}
				}
			}
		}
	case spinner.TickMsg:
		select {
		case m = <-m.loadingFinished:
			m.resetSpinner()
			m.loading = false
			switch m.activeView {
			case activeViewAddNewRepo:
				m.addNewRepoTextInput.Focus()
			}
		default:
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.repos.SetSize(msg.Width-h, msg.Height-v)
		m.efforts.SetSize(msg.Width-h, msg.Height-v)
	case errMsg:
		m.err = msg
		return m, nil
	}

	// if I don't do this down here the updates don't work properly, seems casting to type is causing an issue
	switch m.activeView {
	case activeViewListRepos:
		m.repos, cmd = m.repos.Update(msg)
	case activeViewAddNewRepo:
		m.addNewRepoTextInput, cmd = m.addNewRepoTextInput.Update(msg)
	case activeViewListEfforts:
		m.efforts, cmd = m.efforts.Update(msg)
	case activeViewAddNewEffort:
		m.addNewEffortNameTextInput, cmd = m.addNewEffortNameTextInput.Update(msg)
		m.addNewEffortBranchNameTextInput, cmd = m.addNewEffortBranchNameTextInput.Update(msg)
	}
	return m, cmd
}
