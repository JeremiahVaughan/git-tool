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
	case activeViewAddNewEffort:
		display = fmt.Sprintf(
			"Add an effort\n%s\n",
			m.addNewEffortTextInput.View(),
		)
	case activeViewEditEffort:
		var availableRepos []string
		for i, theRepo := range m.effortRepoSelection {
			var selectedMarker string
			if theRepo.Selected {
				selectedMarker = "[x]"
			} else {
				selectedMarker = "[ ]"
			}
			repoTitle := theRepo.Title()
			if m.listFilterLive || m.listFilterSet {
				repoTitle = highlightFoundText(repoTitle, m.listFilterTextInput.Value())
			}
			itemDisplay := fmt.Sprintf("  %s %s", selectedMarker, repoTitle)
			if m.cursor == i && !m.listFilterLive {
				itemDisplay = lipgloss.NewStyle().Foreground(lipgloss.Color("201")).Render(itemDisplay)
			}
			availableRepos = append(availableRepos, itemDisplay)
		}
		title := fmt.Sprintf("Add repos to \"%s\"\n", m.selectedEffort.Desc)
		title += m.listFilterTextInput.View()
		display = fmt.Sprintf(
			"%s\n%s",
			title,
			strings.Join(availableRepos, "\n"),
		)
	case activeViewListRepos:
		display = docStyle.Render(m.repos.View())
	case activeViewListEfforts:
		display = docStyle.Render(m.efforts.View())
	}

	if m.validationMsg != "" {
		display += getErrorStyle(m.validationMsg)
	}

	return display
}

func getErrorStyle(errMsg string) string {
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true).Width(80)
	return fmt.Sprintf("\n%v", errorStyle.Render(errMsg))
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
