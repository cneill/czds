package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

var (
	configFile = "config.json"
	verbose    bool

	downloadFS, listFS, parseFS *flag.FlagSet
)

func init() {
	flag.StringVar(&configFile, "config", "config.json", "config file to load")
	flag.BoolVar(&verbose, "verbose", false, "verbose output")

	var eh = flag.ExitOnError
	downloadFS = flag.NewFlagSet("download", eh)
	listFS = flag.NewFlagSet("list", eh)
	parseFS = flag.NewFlagSet("parse", eh)

	flag.Usage = func() {
		fmt.Println("Usage: czds [GLOBAL OPTIONS] <COMMAND> [COMMAND OPTIONS]")
		fmt.Println("Commands:")
		fmt.Println("* download")
		fmt.Println("* list")
		fmt.Println("* parse")
		fmt.Println("\nGlobal options:")
		flag.PrintDefaults()
		fmt.Println("\nCommand options:")
		downloadFS.PrintDefaults()
		listFS.PrintDefaults()
		parseFS.PrintDefaults()
	}
}

func downloadZoneFiles(conf *Config) error {
	client := NewClient(conf, verbose)
	if err := client.Auth(); err != nil {
		return fmt.Errorf("failed to authenticate: %v", err)
	}

	links, err := client.GetZoneLinks()
	if err != nil {
		return fmt.Errorf("failed to retrieve zone links: %v", err)
	}

	if err := client.DownloadZoneFiles(links); err != nil {
		return fmt.Errorf("error downloading zone files: %v", err)
	}

	return nil
}

func listZoneFiles(conf *Config) error {
	client := NewClient(conf, verbose)
	if err := client.Auth(); err != nil {
		return fmt.Errorf("failed to authenticate: %v", err)
	}

	links, err := client.GetZoneLinks()
	if err != nil {
		return fmt.Errorf("failed to retrieve zone links: %v", err)
	}

	for _, l := range links {
		fmt.Println(l)
	}
	return nil
}

func parseZoneFiles(conf *Config) error {
	return fmt.Errorf("Unimplemented")
}

func main() {
	if len(os.Args) < 2 {
		flag.Usage()
		log.Fatalf("No command provided")
	}

	flag.Parse()

	conf, err := ParseConfig(configFile)
	if err != nil {
		log.Fatalf("Error parsing config: %v", err)
	}

	var cmd = flag.Arg(0)
	var args = []string{}
	if flag.NArg() > 1 {
		args = flag.Args()
	}
	switch cmd {
	case "download":
		downloadFS.Parse(args)
		if err := downloadZoneFiles(conf); err != nil {
			log.Fatalf("Error downloading zone files: %v", err)
		}
	case "list":
		listFS.Parse(args)
		if err := listZoneFiles(conf); err != nil {
			log.Fatalf("Error listing zone files: %v", err)
		}
	case "parse":
		parseFS.Parse(args)
		if err := parseZoneFiles(conf); err != nil {
			log.Fatalf("Error parsing zone files: %v", err)
		}
	default:
		flag.Usage()
		log.Fatalf("Invalid command provided")
	}

}
