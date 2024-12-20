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
			case activeViewDeleteEffort:
				switch msg.Type {
				case tea.KeyEnter:
					if !m.loading {
						required := m.efforts.SelectedItem().(effort).Name
						if m.deleteEffortTextInput.Value() != required {
							m.validationMsg = fmt.Sprintf("Input must match \"%s\"", required)
						} else {
							theEffort := m.efforts.SelectedItem().(effort)
							m.loading = true
							m.deleteEffortTextInput.Blur()
							go func() {
								var md modelData
								err := deleteEffort(theEffort)
								if err != nil {
									md.err = err
								} else {
									md.err = nil
									md.resetControls = true
									md.activeView = activeViewListEfforts
								}
								loadingFinished <- md
							}()
						}
						return m, m.spinner.Tick
					}

				}
			case activeViewDeleteRepo:
				switch msg.Type {
				case tea.KeyEnter:
					if !m.loading {
						required := m.repos.SelectedItem().(repo).Title()
						if m.deleteRepoTextInput.Value() != required {
							m.validationMsg = fmt.Sprintf("Input must match \"%s\"", required)
						} else {
							theRepo := m.repos.SelectedItem().(repo)
							m.loading = true
							m.deleteRepoTextInput.Blur()
							go func() {
								var md modelData
								err := deleteRepo(theRepo)
								if err != nil {
									md.err = err
								} else {
									md.err = nil
									md.resetControls = true
									md.activeView = activeViewListRepos
								}
								loadingFinished <- md
							}()
						}
						return m, m.spinner.Tick
					}

				}
			case activeViewListEfforts:
				if m.repos.FilterState() != list.Filtering {
					if key.Matches(msg, addItemKeyBinding) {
						m.activeView = activeViewAddNewEffort
						return m, cmd
					} else if key.Matches(msg, deleteItemKeyBinding) {
						m.activeView = activeViewDeleteEffort
						m.selectedEffort = m.efforts.SelectedItem().(effort)
						m.deleteEffortTextInput.Focus()
						return m, cmd
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
						theRepoItems, err := fetchEffortRepoChoices(m.selectedEffort.Id, m.repos)
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
						m.activeView = activeViewDeleteRepo
						m.selectedRepo = m.repos.SelectedItem().(repo)
						m.deleteRepoTextInput.Focus()
						return m, cmd
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
							var md modelData
							validationMsg, err := addRepo(m.addNewRepoTextInput.Value())
							if err != nil || validationMsg != "" {
								md.err = err
								md.validationMsg = validationMsg
							} else {
								md.err = nil
								md.validationMsg = ""
								md.resetControls = true
								repos, err := fetchRepos()
								if err != nil {
									md.err = fmt.Errorf("error, when fetchRepos() for Update(). Error: %v", err)
								} else {
									m.repos.SetItems(repos)
									md.repos = m.repos
									md.activeView = activeViewListRepos
								}
							}
							loadingFinished <- md
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
						m.err = fmt.Errorf("error, when fetchEfforts() for Update() after adding effort. Error: %v", err)
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
						if !m.loading {
							m.loading = true
							go func() {
								var md modelData
								validationMsg, err := applyRepoSelectionForEffort(m.selectedEffort, m.repos.Items())
								if err != nil || validationMsg != "" {
									md.err = err
									md.validationMsg = validationMsg
								} else {
									md.err = nil
									md.validationMsg = ""
									md.resetControls = true
									md.activeView = activeViewListEfforts
								}
								loadingFinished <- md
							}()
							return m, m.spinner.Tick
						}
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
		case md := <-loadingFinished:
			m.resetSpinner()
			m.loading = false
			m.err = md.err
			m.validationMsg = md.validationMsg
			switch m.activeView {
			case activeViewAddNewRepo:
				m.addNewRepoTextInput.Focus()
				if md.resetControls {
					m.addNewRepoTextInput.Reset()
				}
				m.repos = md.repos
				m.activeView = md.activeView
			case activeViewEditEffort:
				if md.resetControls {
					m.effortRepoVisibleSelection = resetRepoSelection(m.effortRepoVisibleSelection)
				}
				m.activeView = md.activeView
			case activeViewDeleteEffort:
				if md.resetControls {
					m.deleteEffortTextInput.Reset()
				}
				m.activeView = md.activeView
				efforts, err := fetchEfforts()
				if err != nil {
					m.err = fmt.Errorf("error, when fetchEfforts() for Update() after deleting effort. Error: %v", err)
					return m, cmd
				}
				m.efforts.SetItems(efforts)
			case activeViewDeleteRepo:
				if md.resetControls {
					m.deleteRepoTextInput.Reset()
				}
				m.activeView = md.activeView
				repos, err := fetchRepos()
				if err != nil {
					m.err = fmt.Errorf("error, when fetchRepos() for Update() after deleting repo. Error: %v", err)
					return m, cmd
				}
				m.repos.SetItems(repos)
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
	case activeViewDeleteEffort:
		m.deleteEffortTextInput, cmd = m.deleteEffortTextInput.Update(msg)
	case activeViewDeleteRepo:
		m.deleteRepoTextInput, cmd = m.deleteRepoTextInput.Update(msg)
	}
	return m, cmd
}
