package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
)

func init() {
	if os.Getenv("TEST_MODE") != "true" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("error, could not find the home directory. Error: %v", err)
		}
		dataDirectory = fmt.Sprintf("%s/git_tool_data/", homeDir)
		reposDirectory = dataDirectory + "repos/"
		effortsDirectory = dataDirectory + "efforts/"
		err = os.MkdirAll(dataDirectory, os.ModePerm)
		if err != nil {
			log.Fatalf("error, could not create data directory. Error: %v", err)
		}

		dbFile := fmt.Sprintf("%s%s", dataDirectory, "data")
		_, err = os.Stat(dbFile)
		if os.IsNotExist(err) {
			var file *os.File
			file, err = os.Create(dbFile)
			if err != nil {
				log.Fatalf("error, when creating db file. Error: %v", err)
			}
			file.Close()
		} else if err != nil {
			// An error other than the file not existing occurred
			log.Fatalf("error, when checking db file exists. Error: %v", err)
		}

		database, err = sql.Open("sqlite3", dbFile)
		if err != nil {
			log.Fatalf("error, when establishing connection with sqlite db. Error: %v", err)
		}
	}
}
