package feeds2imap

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchNewFeedItems(t *testing.T) {
	testInit()
	defer testTearDown()

	require.NotZero(t, len(FetchNewFeedItems()))
}

func TestFetchNewFeedItemsWithCommit(t *testing.T) {
	testInit()
	defer testTearDown()

	items := FetchNewFeedItems()
	CommitToCache(items)

	require.Zero(t, len(FetchNewFeedItems()))
}
