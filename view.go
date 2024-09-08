package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	switch m.activeView {
	case activeViewAddNewRepo:
		display := fmt.Sprintf(
			"Add a repo\n%s\n",
			m.addNewRepo.View(),
		)
		if m.err != nil {
			errorStyle := getErrorStyle()
			errMsg := errorStyle.Render(m.err.Error())
			display += fmt.Sprintf("\n%v", errMsg)
		}
		if m.validationMsg != "" {
			errorStyle := getErrorStyle()
			display += fmt.Sprintf("\n%v", errorStyle.Render(m.validationMsg))
		}
		return display
	case activeViewListEfforts:
		// display := fmt.Sprintf(
		// )
	}

	return docStyle.Render(m.repos.View())
}

func getErrorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true).Width(80)
}
