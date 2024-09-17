package main

import (
	"strings"
	"testing"
)

func Test_generateInsertStatement(t *testing.T) {
	repos := []repo{
		{Id: 1},
		{Id: 2},
		{Id: 3},
	}
	got := generateInsertStatement(repos)
	expected := `INSERT OR IGNORE INTO effort_repo (effort_id, repo_id)
		VALUES (?, ?), (?, ?), (?, ?)`
	gotNoSpaces := strings.ReplaceAll(got, " ", "")
	expectedNoSpaces := strings.ReplaceAll(expected, " ", "")
	if gotNoSpaces != expectedNoSpaces {
		t.Errorf("got %s, but wanted %s", got, expected)
	}
}
