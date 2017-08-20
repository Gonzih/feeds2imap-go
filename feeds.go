package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mmcdole/gofeed"
	"github.com/spf13/viper"
)

// InputURLs represents freshly parsed configuration
type InputURLs map[string][]string

// FlatURLs represents url -> folder map
type FlatURLs map[string]string

// FeedWithFolder represents folder name and feed item combined
type FeedWithFolder struct {
	Feed   *gofeed.Feed
	Folder string
}

// FeedsWithFolders represents collection of FeedWithFolder
type FeedsWithFolders []FeedWithFolder

// ItemsWithFolders represents rss item (post), folder and original feed title cobined
type ItemWithFolder struct {
	Item      *gofeed.Item
	Folder    string
	FeedTitle string
	FeedLink  string
}

// ItemsWithFolders represents collection of ItemWithFolder
type ItemsWithFolders []ItemWithFolder

// ItemsCache represents GUIDs cache
type ItemsCache []string

func readInputURLsFile() InputURLs {
	return InputURLs(viper.GetStringMapStringSlice("urls"))
}

func flattenInputURLs(urls InputURLs) FlatURLs {
	flaturls := make(FlatURLs)

	for folder, links := range urls {
		for _, link := range links {
			flaturls[link] = folder
		}
	}

	return flaturls
}

func fetchFeedData(urls FlatURLs) (FeedsWithFolders, error) {
	var parsedLock sync.Mutex
	var wg sync.WaitGroup
	var parsed FeedsWithFolders

	for url, folder := range urls {
		wg.Add(1)

		go func(url, folder string) {
			defer wg.Done()

			if viper.GetBool("debug") {
				log.Printf("Fetching: %s", url)
			}

			fp := gofeed.NewParser()
			feed, err := fp.ParseURL(url)

			if err != nil {
				log.Printf("Error while fetching %s: %s", url, err)
				return
			}

			parsedLock.Lock()
			defer parsedLock.Unlock()
			parsed = append(parsed, FeedWithFolder{Feed: feed, Folder: folder})

		}(url, folder)
	}

	wg.Wait()

	return parsed, nil
}

func flattenFeedData(feeds FeedsWithFolders) (items ItemsWithFolders) {
	for _, fWithFolder := range feeds {
		folder := fWithFolder.Folder
		for _, item := range fWithFolder.Feed.Items {
			items = append(items, ItemWithFolder{Item: item, Folder: folder, FeedTitle: fWithFolder.Feed.Title, FeedLink: fWithFolder.Feed.Link})
		}
	}

	return
}

// ReadCacheFile reads cache file from fs
func ReadCacheFile() ItemsCache {
	var cache ItemsCache

	fname := viper.GetString("paths.cache")

	if _, err := os.Stat(fname); os.IsNotExist(err) {
		return cache
	}

	f, err := os.Open(fname)

	if err != nil {
		log.Println(err)
		return cache
	}

	bytes, err := ioutil.ReadAll(f)

	if err != nil {
		log.Println(err)
		return cache
	}

	err = json.Unmarshal(bytes, &cache)

	return cache
}

// CommitToCache saves item data to db
func CommitToCache(items ItemsWithFolders) error {
	for _, item := range items {
		i := item.Item

		uuid := uuid.New().String()
		author := formatAuthor(i)
		link := formatLink(i.Link)

		var content string
		if len(i.Content) > 0 {
			content = i.Content
		} else {
			content = i.Description
		}

		var published time.Time
		if i.PublishedParsed != nil {
			published = *i.PublishedParsed
		} else {
			published = time.Now()
		}

		err := CommitToDB(uuid, i.GUID, i.Title, link, author, item.FeedTitle, item.FeedLink, item.Folder, content, published)

		if err != nil {
			return err
		}
	}

	return nil
}

func filterNewItems(entries ItemsWithFolders) (newItems ItemsWithFolders) {
	for _, entry := range entries {
		if IsExistingID(entry.Item.GUID) {
			continue
		}

		newItems = append(newItems, entry)
	}

	return
}

// FetchNewFeedItems loads configuration, fetches rss items and discards ones that are in cache already returning new items and new version of a cache
func FetchNewFeedItems() ItemsWithFolders {
	input := readInputURLsFile()

	flat := flattenInputURLs(input)

	parsed, err := fetchFeedData(flat)

	if err != nil {
		log.Fatal(err)
	}

	allItems := flattenFeedData(parsed)
	return filterNewItems(allItems)
}
