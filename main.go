package main

import (
	"flag"
	"fmt"
	"github.com/newrelic/go-agent"
	"os"
)

func main() {
	var (
		configFile string
	)

	flag.StringVar(&configFile, "config", "./config.json", "Config filepath")
	flag.Parse()

	configuration := &Configuration{}
	configuration.Init(configFile)

	_, err := newrelic.NewApplication(
		newrelic.NewConfig("WebSocketerD", configuration.NewRelicLicenseKey),
	)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	server := newServer(configuration)
	server.Run()
}
