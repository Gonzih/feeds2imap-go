package feeds2imap

import (
	"bytes"
	"fmt"
	"time"

	"github.com/emersion/go-imap"
)

type mockImapMessage struct {
	date time.Time
	msg  string
}

type mockImapClient struct {
	messages map[string][]string
	folders  map[string]bool
}

func newMockImapClient() *mockImapClient {
	return &mockImapClient{
		messages: make(map[string][]string, 0),
		folders:  make(map[string]bool, 0),
	}
}

func (mic *mockImapClient) Logout() error {
	return nil
}

func (mic *mockImapClient) Create(folder string) error {
	mic.folders[folder] = true
	return nil
}

func (mic *mockImapClient) Append(mbox string, flags []string, date time.Time, msg imap.Literal) error {
	if !mic.folders[mbox] {
		return fmt.Errorf("Folder %s was not found", mbox)
	}

	buf := bytes.NewBuffer([]byte{})
	buf.ReadFrom(msg)
	mic.messages[mbox] = append(mic.messages[mbox], buf.String())
	return nil
}
