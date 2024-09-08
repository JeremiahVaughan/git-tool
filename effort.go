package main


type effort struct {
	name string
	discription string
	repos []repo
}

func (e effort) Title() string {
	return e.name
}
func (e effort) Description() string { return e.Description() }
func (e effort) FilterValue() string { return e.name }
