package main

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
)

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", viper.GetString("paths.db"))

	if err != nil {
		log.Fatal(err)
	}
}

func migrateDB() {
	sqlStmt := `
	CREATE TABLE IF NOT EXISTS feeds (uuid STRING NOT NULL PRIMARY KEY, guid STRING, title STRING, link STRING, author STRING, feedtitle STRING, feedlink STRING, folder STRING, content TEXT, published_at TIMESTAMP, read BOOL);
	CREATE INDEX IF NOT EXISTS guid_index ON feeds (guid);
	CREATE INDEX IF NOT EXISTS folder_index ON feeds (folder);
	CREATE INDEX IF NOT EXISTS published_index ON feeds (published_at);
	CREATE INDEX IF NOT EXISTS read_index ON feeds (read);
	`

	_, err := db.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("%q: %s\n", err, sqlStmt)
	}
}

func closeDB() {
	if db != nil {
		db.Close()
	}
}

// IsExistingID tries to find maching id in db
func IsExistingID(guid string) bool {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM feeds WHERE guid=?;", guid).Scan(&count)

	if err != nil {
		log.Printf("Error scanning row: %s", err)
		return false
	}

	if count > 0 {
		return true
	}

	return false
}

// CommitToDB stores entry in the db
func CommitToDB(uuid, guid, title, link, author, feedtitle, feedlink, folder, content string, time time.Time) error {
	_, err := db.Exec("INSERT INTO feeds VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);", uuid, guid, title, link, author, feedtitle, feedlink, folder, content, time, false)
	return err
}

// ScanRowToItem scans a single row into a struct
func ScanRowToItem(rows *sql.Rows) (i dbFeedItem, err error) {
	err = rows.Scan(&i.UUID, &i.GUID, &i.Title, &i.Link, &i.Author, &i.FeedTitle, &i.FeedLink, &i.Folder, &i.Content, &i.Published, &i.Read)

	return i, err
}

// MarkAsReadInDB sets read field to true on the given id
func MarkAsReadInDBByID(uuid string) error {
	_, err := db.Exec("UPDATE feeds SET read = 1 WHERE uuid = ?;", uuid)

	return err
}

func MarkAsReadInDBByFolder(folder string) error {
	_, err := db.Exec("UPDATE feeds SET read = 1 WHERE folder = ?;", folder)

	return err
}
