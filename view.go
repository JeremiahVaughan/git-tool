package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	if m.err != nil {
		return getErrorStyle(m.err.Error())
	}

	var display string
	switch m.activeView {
	case activeViewAddNewRepo:
		display = fmt.Sprintf(
			"Add a repo\n%s\n",
			m.addNewRepo.View(),
		)
	case activeViewAddNewEffort:
		display = fmt.Sprintf(
			"Add an effort\n%s\n",
			m.addNewEffort.View(),
		)
	case activeViewEditEffort:
		// todo add filter mode
		var availableRepos []string
		for i, r := range m.repos.Items() {
			theRepo := r.(repo)
			var selectedMarker string
			if m.effortRepoSelection[i] {
				selectedMarker = "[x]"
			} else {
				selectedMarker = "[ ]"
			}
			itemDisplay := fmt.Sprintf("  %s %s", selectedMarker, theRepo.Title())
			if m.cursor == i {
				itemDisplay = lipgloss.NewStyle().Foreground(lipgloss.Color("201")).Render(itemDisplay)
			}
			availableRepos = append(availableRepos, itemDisplay)
		}
		display = fmt.Sprintf(
			"Add repos to \"%s\"\n%s\n",
			m.selectedEffort.Desc,
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
