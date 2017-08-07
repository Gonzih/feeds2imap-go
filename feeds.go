package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"sync"

	"github.com/mmcdole/gofeed"
	"github.com/spf13/viper"
)

type InputURLs map[string][]string
type FlatURLs map[string]string

type FeedWithFolder struct {
	Feed   *gofeed.Feed
	Folder string
}
type FeedsWithFolders []FeedWithFolder

type ItemWithFolder struct {
	Item      *gofeed.Item
	Folder    string
	FeedTitle string
}
type ItemsWithFolders []ItemWithFolder

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
			items = append(items, ItemWithFolder{Item: item, Folder: folder, FeedTitle: fWithFolder.Feed.Title})
		}
	}

	return
}

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

func WriteCacheFile(cache ItemsCache) error {
	json, err := json.Marshal(&cache)

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(viper.GetString("paths.cache"), json, 0644)

	if err != nil {
		return err
	}

	return nil
}

func filterNewItems(entries ItemsWithFolders) (newItems ItemsWithFolders, newCache ItemsCache) {
	cache := ReadCacheFile()
	newCache = cache

OUTER:
	for _, entry := range entries {
		for _, cacheEntry := range cache {
			if cacheEntry == entry.Item.GUID {
				continue OUTER
			}
		}

		newItems = append(newItems, entry)
		newCache = append(newCache, entry.Item.GUID)
	}

	return
}

func FetchNewFeedItems() (ItemsWithFolders, ItemsCache) {
	input := readInputURLsFile()

	flat := flattenInputURLs(input)

	parsed, err := fetchFeedData(flat)

	if err != nil {
		log.Fatal(err)
	}

	allItems := flattenFeedData(parsed)
	return filterNewItems(allItems)
}
