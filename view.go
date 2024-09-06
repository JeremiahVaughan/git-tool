package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)

func (m model) View() string {
	if m.activeView == activeViewAddNewRepo {
		display := fmt.Sprintf(
			"Add a repo\n%s\n",
			m.addNewRepo.View(),
		)
		if m.err != nil {
			errMsg := errorStyle.Render(m.err.Error())
			display += fmt.Sprintf("\n%v", errMsg)
		}
		if m.validationMsg != "" {
			display += fmt.Sprintf("\n%v", errorStyle.Render(m.validationMsg))
		}
		return display
	}

	return docStyle.Render(m.repos.View())
}
