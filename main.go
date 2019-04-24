package main

import (
	"log"
	"time"

	"github.com/Gonzih/feeds2imap-go/lib"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func init() {
	pflag.String("config", "config.yaml", "config file path")

	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	viper.SetConfigFile(viper.GetString("config"))

	err := viper.ReadInConfig()

	if err != nil {
		panic(err)
	}

	feeds2imap.InitDB()
	feeds2imap.MigrateDB()
}

func main() {
	defer feeds2imap.CloseDB()

	for {
		items := feeds2imap.FetchNewFeedItems()

		if len(items) > 0 {
			if viper.GetBool("debug") {
				log.Printf("Found %d new items", len(items))
			}

			if viper.GetBool("imap.enabled") {
				err := feeds2imap.AppendNewItemsViaIMAP(items)

				if err != nil {
					log.Fatal(err)
				}
			}

			err := feeds2imap.CommitToCache(items)

			if err != nil {
				log.Fatal(err)
			}
		}

		if !viper.GetBool("daemon.enabled") {
			break
		} else {
			t := viper.GetInt("daemon.delay")

			if viper.GetBool("debug") {
				log.Printf("Sleeping in a loop for %d minutes", t)
			}

			time.Sleep(time.Minute * time.Duration(t))
		}
	}
}
