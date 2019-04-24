package feeds2imap

import (
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	// go-sqlite3 imports sqlite3 driver for sql package
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
)

type dbFeedItem struct {
	UUID      string    `db:"uuid"`
	GUID      string    `db:"guid"`
	Title     string    `db:"title"`
	Link      string    `db:"link"`
	Author    string    `db:"author"`
	FeedTitle string    `db:"feedtitle"`
	FeedLink  string    `db:"feedlink"`
	Folder    string    `db:"folder"`
	Published time.Time `db:"published"`
}

var db *sqlx.DB

// InitDB will init sqlx sqlite3 db
func InitDB() {
	var err error
	db, err = sqlx.Open("sqlite3", viper.GetString("paths.db"))

	if err != nil {
		log.Fatal(err)
	}
}

// MigrateDB will create test db
func MigrateDB() {
	sqlStmt := `
	CREATE TABLE IF NOT EXISTS feeds (uuid STRING NOT NULL PRIMARY KEY, guid STRING, title STRING, link STRING, author STRING, feedtitle STRING, feedlink STRING, folder STRING, published_at TIMESTAMP);
	CREATE INDEX IF NOT EXISTS guid_index ON feeds (guid);
	`

	db.MustExec(sqlStmt)
}

// CloseDB will close sqlx db
func CloseDB() {
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
func CommitToDB(item *dbFeedItem) error {
	_, err := db.NamedExec("INSERT INTO feeds VALUES (:uuid,:guid,:title,:link,:author,:feedtitle,:feedlink,:folder,:published);", item)
	return err
}
