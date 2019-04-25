package feeds2imap

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppendItems(t *testing.T) {
	testInit()
	defer testTearDown()

	items := FetchNewFeedItems()
	client := newMockImapClient()
	appendNewItemsVia(items, client)
	require.True(t, client.folders["TEST-RSS/go"])
	require.False(t, client.folders["TEST-RSS/non-existant-folder"])
	require.Len(t, client.messages["TEST-RSS/go"], len(items))
}
