package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stevenstank/bolt/internal/command"
	"github.com/stevenstank/bolt/internal/engine"
	"github.com/stevenstank/bolt/internal/pubsub"
	"github.com/stevenstank/bolt/internal/replication"
	"github.com/stevenstank/bolt/internal/server"
	"github.com/stevenstank/bolt/internal/storage"
)

type config struct {
	Addr         string
	AOFPath      string
	SnapshotPath string
	ReplicaOf    string
}

func main() {
	config, err := parseConfig(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	store, err := storage.NewDurableStore(config.AOFPath, config.SnapshotPath)
	if err != nil {
		log.Fatal(err)
	}

	var replica *replication.Replica
	var primary *replication.Primary
	db := engine.New(store)
	if config.ReplicaOf != "" {
		db.SetReadOnly(true)
		replica = replication.NewReplica(replication.ReplicaConfig{
			PrimaryAddr: config.ReplicaOf,
			Store:       db,
		}, nil)
	} else {
		primary = replication.NewPrimary(store, nil)
		db = engine.New(store, primary)
	}

	var replicaAccepter server.ReplicaAccepter
	if primary != nil {
		replicaAccepter = primary
	}

	pubsubHub := pubsub.NewHub()

	var replicationInfo server.ReplicationInfo

if primary != nil {
	replicationInfo = primary
}

if replica != nil {
	replicationInfo = replica
}

srv := server.New(server.Config{
	Addr:            config.Addr,
	Processor:       command.NewProcessor(command.NewDispatcher(db)),
	ReplicaAccepter: replicaAccepter,
	ReplicationInfo: replicationInfo,
	PubsubHub:       pubsubHub,
	Role:            getRole(config.ReplicaOf),
})
	srv.SetProcessorInfo()
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}

	replicaCtx, cancelReplica := context.WithCancel(context.Background())
	defer cancelReplica()
	if replica != nil {
		go replica.Run(replicaCtx)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	<-signals

	cancelReplica()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
	if err := store.SaveSnapshot(); err != nil {
		log.Fatal(err)
	}
}

func parseConfig(args []string) (config, error) {
	flags := flag.NewFlagSet("bolt", flag.ContinueOnError)

	var config config
	flags.StringVar(&config.Addr, "addr", "127.0.0.1:6379", "TCP listen address")
	flags.StringVar(&config.AOFPath, "aof", "bolt.aof", "append-only persistence file")
	flags.StringVar(&config.SnapshotPath, "snapshot", "bolt.snapshot", "snapshot persistence file")
	flags.StringVar(&config.ReplicaOf, "replicaof", "", "primary address to replicate from")

	if err := flags.Parse(args); err != nil {
		return config, err
	}
	return config, nil
}

func getRole(replicaOf string) string {
	if replicaOf != "" {
		return "replica"
	}
	return "primary"
}
