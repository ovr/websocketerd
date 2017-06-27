package main

import (
	"flag"
	"fmt"
	"github.com/newrelic/go-agent"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"time"
)

func main() {
	var (
		configFile string
	)

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	flag.StringVar(&configFile, "config", "./config.json", "Config filepath")
	flag.Parse()

	configuration := &Configuration{}
	configuration.Init(configFile)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	app, err := newrelic.NewApplication(
		newrelic.NewConfig(configuration.NewRelic.AppName, configuration.NewRelic.Key),
	)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	server := newServer(configuration, app)
	server.Run()

	<-stop

	log.Println("Shutting down the server...")

	server.Shutdown()
	app.Shutdown(10 * time.Second)
}
