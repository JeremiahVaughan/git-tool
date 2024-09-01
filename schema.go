package main

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var DatabaseMigrationDirectory = "database"
var database *sql.DB

func init() {
	if os.Getenv("TEST_MODE") == "false" {
		log.Printf("initializing schema")
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("error, could not find the home directory. Error: %v", err)
		}
		gitToolData := fmt.Sprintf("%s/git_tool_data/", homeDir)
		err = os.MkdirAll(gitToolData, os.ModePerm)
		if err != nil {
			log.Fatalf("error, could not create data directory. Error: %v", err)
		}

		log.Printf("file connection")
		dbFile := fmt.Sprintf("%s%s", gitToolData, "data")
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

		log.Printf("openning connection")
		database, err = sql.Open("sqlite3", dbFile)
		if err != nil {
			log.Fatalf("error, when establishing connection with sqlite db. Error: %v", err)
		}
		log.Printf("out connection")
	}
}

func ProcessSchemaChanges(databaseFiles embed.FS) error {
	err := createInitTable()
	if err != nil {
		return fmt.Errorf("error occurred when attempting to create init table: %v", err)
	}

	log.Println("checking for migrations ...")
	dirEntries, err := fs.ReadDir(databaseFiles, DatabaseMigrationDirectory)
	if err != nil {
		return fmt.Errorf("an error has occurred when attempting to read database directory. Error: %v", err)
	}
	var migrationFileCandidateFileNames []string
	for _, entry := range dirEntries {
		if !entry.IsDir() {
			migrationFileCandidateFileNames = append(migrationFileCandidateFileNames, entry.Name())
		}
	}

	migrationFiles := filterForMigrationFiles(migrationFileCandidateFileNames)
	var migrationsCompleted []string
	noMigrationsToProcessMessage := "no database migration files to process, skipping migrations ..."
	if len(migrationFiles) == 0 {
		log.Println(noMigrationsToProcessMessage)
		return nil
	} else {
		migrationsCompleted, err = checkForCompletedMigrations()
		if err != nil {
			return fmt.Errorf("error has occurred when checking for completed migrations: %v", err)
		}
	}

	migrationsNeeded := determineMigrationsNeeded(migrationFiles, migrationsCompleted)
	migrationsNeededSorted := sortMigrationsNeededFiles(migrationsNeeded)
	for _, fileName := range migrationsNeededSorted {
		log.Printf("attempting to perform database migration with %s ...", fileName)

		filePath := fmt.Sprintf("%s/%s", DatabaseMigrationDirectory, fileName)
		err = executeSQLFile(filePath, databaseFiles)
		if err != nil {
			return fmt.Errorf("error occurred when executing sql script: Filename: %s. Error: %v", fileName, err)
		}
		err = recordSuccessfulMigration(fileName)
		if err != nil {
			return fmt.Errorf("error has occurred when attempting to record a successful migration: %v", err)
		}
	}
	log.Println("finished database schema changes")
	return nil
}

func createInitTable() error {
	_, err := database.Exec(`EXISTS init
                        (
                            id  SERIAL NOT NULL
                                CONSTRAINT init_pk 
                                    PRIMARY KEY,
                            migration_file_name TEXT   NOT NULL
                                CONSTRAINT init_migration_file_name_uindex
                                    UNIQUE
                        )`)
	if err != nil {
		return fmt.Errorf("error, when executing query to create init table: %v", err)
	}
	return nil
}

func sortMigrationsNeededFiles(needed []string) []string {
	re := regexp.MustCompile(`^(\d+)`)

	sort.Slice(needed, func(i, j int) bool {
		num1, _ := strconv.Atoi(re.FindStringSubmatch(needed[i])[1])
		num2, _ := strconv.Atoi(re.FindStringSubmatch(needed[j])[1])
		return num1 < num2
	})
	return needed
}

func determineMigrationsNeeded(migrationFiles []string, migrationsCompleted []string) []string {
	var migrationsNeeded []string
	migrationsCompletedMap := make(map[string]bool)
	for _, value := range migrationsCompleted {
		migrationsCompletedMap[value] = true
	}
	for _, value := range migrationFiles {
		if !migrationsCompletedMap[value] {
			migrationsNeeded = append(migrationsNeeded, value)
		}
	}
	return migrationsNeeded
}

func filterForMigrationFiles(candidates []string) []string {
	var migrationFiles []string
	re := regexp.MustCompile(`^\d+`)
	for _, fileName := range candidates {
		if re.MatchString(fileName) {
			migrationFiles = append(migrationFiles, fileName)
		}
	}
	return migrationFiles
}

func recordSuccessfulMigration(fileName string) error {
	_, err := database.Exec(
		"INSERT INTO init (migration_file_name)\nVALUES (?)",
		fileName,
	)
	if err != nil {
		return fmt.Errorf("error occurred when attempting to run sql command: %v", err)
	}
	return nil
}

func checkForCompletedMigrations() (results []string, err error) {
	rows, err := database.Query(
		"SELECT migration_file_name\nFROM init",
	)
	defer func() {
		err = rows.Err()
		if err != nil {
			err = fmt.Errorf("error, occurred when reading rows. Error: %v", err)
		}
		rows.Close()
	}()

	if err != nil {
		return nil, fmt.Errorf("error has occurred when attempting to retrieve pending migrations: %v", err)
	}

	for rows.Next() {
		var result string
		err = rows.Scan(
			&result,
		)
		if err != nil {
			return nil, fmt.Errorf("error has occurred when scanning for pending migrations: %v", err)
		}
		results = append(results, result)
	}

	return results, nil
}

func executeSQLFile(filePath string, databaseFiles embed.FS) error {
	content, err := databaseFiles.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read SQL file: %w", err)
	}

	sql := string(content)
	queries := strings.Split(sql, ";")

	for _, query := range queries {
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}

		_, err = database.Exec(query)
		if err != nil {
			return fmt.Errorf("error, failed to execute QUERY: %s. ERROR: %v", query, err)
		}
	}

	return nil
}
