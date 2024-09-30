package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var highlightStyle = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("226")).Foreground(lipgloss.Color("#000000"))

func (m model) View() string {
	if m.err != nil {
		return getErrorStyle(m.err.Error())
	}

	var display string
	switch m.activeView {
	case activeViewAddNewRepo:
		display = fmt.Sprintf(
			"Add a repo\n%s\n",
			m.addNewRepoTextInput.View(),
		)
		if m.loading {
			display += m.spinner.View()
		}
	case activeViewAddNewEffort:
		display = fmt.Sprintf(
			"Add an effort\n\nEffort name\n%s\n\nBranch Name\n%s\n",
			m.addNewEffortNameTextInput.View(),
			m.addNewEffortBranchNameTextInput.View(),
		)
	case activeViewEditEffort:
		var availableRepos []string
		for i, theRepo := range m.effortRepoVisibleSelection {
			var selectedMarker string
			if theRepo.Selected {
				selectedMarker = "[x]"
			} else {
				selectedMarker = "[ ]"
			}
			repoTitle := theRepo.Title()
			if m.listFilterLive || (m.listFilterSet && m.cursor != i) {
				repoTitle = highlightFoundText(repoTitle, m.listFilterTextInput.Value())
			}
			itemDisplay := fmt.Sprintf("%s %s", selectedMarker, repoTitle)
			itemDisplay = lipgloss.NewStyle().MarginLeft(2).Render(itemDisplay)
			if m.cursor == i && !m.listFilterLive {
				itemDisplay = lipgloss.NewStyle().Foreground(lipgloss.Color("201")).Render(itemDisplay)
			}
			availableRepos = append(availableRepos, itemDisplay)
		}
		title := fmt.Sprintf("Add repos to \"%s\"", m.selectedEffort.Desc)
		title = lipgloss.NewStyle().
			Background(lipgloss.Color("#7d34eb")).
			Foreground(lipgloss.Color("#DDDDDD")).
			Padding(1).
			Render(title)
		textInput := lipgloss.NewStyle().Padding(1).Render(m.listFilterTextInput.View())
		display = fmt.Sprintf(
			"%s\n%s\n%s",
			title,
			textInput,
			strings.Join(availableRepos, "\n"),
		)
	case activeViewListRepos:
		display = m.repos.View()
	case activeViewListEfforts:
		display = m.efforts.View()
	}

	if m.validationMsg != "" {
		display += getErrorStyle(m.validationMsg)
	}
	return docStyle.Render(display)
}

func getErrorStyle(errMsg string) string {
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true).Width(80).MarginLeft(4)
	return fmt.Sprintf("\n\n%v", errorStyle.Render(errMsg))
}

func highlightFoundText(str string, substr string) string {
	if !strings.Contains(str, substr) {
		return str
	}

	prefixAndSuffix := strings.SplitN(str, substr, 2)
	before := prefixAndSuffix[0]
	after := prefixAndSuffix[1]

	highlightedSubstr := highlightStyle.Render(substr)

	return before + highlightedSubstr + after
}
