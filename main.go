package main

import "log"

func main() {
	configFile := "config.json"
	conf, err := ParseConfig(configFile)
	if err != nil {
		log.Fatalf("error parsing config: %v", err)
	}
	client := NewClient(conf)
	if err := client.Auth(); err != nil {
		log.Fatalf("failed to authenticate: %v", err)
	}

	links, err := client.GetZoneLinks()
	if err != nil {
		log.Fatalf("failed to retrieve zone links: %v", err)
	}

	if err := client.DownloadZoneFiles(links); err != nil {
		log.Fatalf("error downloading zone files: %v", err)
	}
}
