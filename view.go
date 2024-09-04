package main

import "fmt"

func (m model) View() string {
	if m.activeView == activeViewAddNewRepo {
		return fmt.Sprintf(
			"Add a repo\n%s\n",
			m.addNewRepo.View(),
		)
	}

	return docStyle.Render(m.repos.View())
}
