package main

import (
	"flag"
	"fmt"
	"github.com/newrelic/go-agent"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"time"
)

func main() {
	var (
		configFile string
	)

	flag.StringVar(&configFile, "config", "./config.json", "Config filepath")
	flag.Parse()

	configuration := &Configuration{}
	configuration.Init(configFile)

	log.SetFormatter(&log.JSONFormatter{})

	if configuration.Debug {
		log.SetLevel(log.DebugLevel)
	}

	app, err := newrelic.NewApplication(
		newrelic.NewConfig(configuration.NewRelic.AppName, configuration.NewRelic.Key),
	)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	server := newServer(configuration, app)
	server.Run()

	<-stop

	log.Println("Shutting down the server...")

	server.Shutdown()
	app.Shutdown(10 * time.Second)
}
