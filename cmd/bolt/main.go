package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stevenstank/bolt/internal/server"
)

type config struct {
	Addr string
}

func main() {
	config, err := parseConfig(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	srv := server.New(server.Config{Addr: config.Addr})
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	<-signals

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
}

func parseConfig(args []string) (config, error) {
	flags := flag.NewFlagSet("bolt", flag.ContinueOnError)

	var config config
	flags.StringVar(&config.Addr, "addr", "127.0.0.1:6379", "TCP listen address")

	if err := flags.Parse(args); err != nil {
		return config, err
	}
	return config, nil
}
