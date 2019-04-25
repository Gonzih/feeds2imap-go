package feeds2imap

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func initViper() {
	viper.SetDefault("paths.db", "test.db")
	viper.SetDefault("urls", map[string]string{
		"go": "https://blog.golang.org/feed.atom",
	})
	viper.SetDefault("imap.folder.prefix", "TEST-RSS")
}

func testInit() {
	initViper()
	InitDB()
	MigrateDB()
}

func testTearDown() {
	CloseDB()
	os.Remove(viper.GetString("paths.db"))
}

func TestInitAndMigrateDB(t *testing.T) {
	testInit()
	defer testTearDown()
}

func TestCommitToDB(t *testing.T) {
	testInit()
	defer testTearDown()

	i := &dbFeedItem{}
	i.UUID = "my-uuid"
	i.GUID = "my-guid"
	err := CommitToDB(i)
	require.Nil(t, err)
}

func TestCommitAndIsExistingIDToDB(t *testing.T) {
	testInit()
	defer testTearDown()

	i := &dbFeedItem{}
	i.UUID = "my-uuid"
	i.GUID = "my-guid"
	err := CommitToDB(i)
	require.Nil(t, err)
	require.True(t, IsExistingID(i.GUID))
}
