package feeds2imap

import (
	"time"
)

type dbFeedItem struct {
	UUID      string    `json:"uuid"`
	GUID      string    `json:"guid"`
	Title     string    `json:"title"`
	Link      string    `json:"link"`
	Author    string    `json:"author"`
	FeedTitle string    `json:"feedtitle"`
	FeedLink  string    `json:"feedlink"`
	Folder    string    `json:"folder"`
	Content   string    `json:"content"`
	Published time.Time `json:"published"`
	Read      bool      `json:"read"`
}

type dbFolderWithCount struct {
	Folder string `json:"folder"`
	Count  string `json:"unread"`
}
