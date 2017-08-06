package main

import (
	"log"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func init() {
	pflag.String("config", "config.yaml", "config file path")
	pflag.Bool("daemon", false, "run forever in a loop")

	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	viper.SetConfigFile(viper.GetString("config"))

	err := viper.ReadInConfig()

	if err != nil {
		panic(err)
	}
}

func main() {
	for {
		items, cache := FetchNewFeedItems()

		if len(items) > 0 {
			err := AppendNewItemsViaIMAP(items)

			if err != nil {
				log.Fatal(err)
			}
		}

		err := WriteCacheFile(cache)

		if err != nil {
			log.Fatal(err)
		}

		if !viper.GetBool("daemon") {
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
