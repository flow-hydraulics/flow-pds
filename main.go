package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/flow-hydraulics/flow-pds/service/app"
	"github.com/flow-hydraulics/flow-pds/service/config"
	"github.com/flow-hydraulics/flow-pds/service/errors"
	pds_http "github.com/flow-hydraulics/flow-pds/service/http"
	"github.com/flow-hydraulics/flow-pds/service/store"
	"github.com/onflow/flow-go-sdk/client"
	"google.golang.org/grpc"
)

const version = "0.1.0"

var (
	sha1ver   string // sha1 revision used to build the program
	buildTime string // when the executable was built
)

func main() {
	var (
		printVersion bool
		envFilePath  string
	)

	// If we should just print the version number and exit
	flag.BoolVar(&printVersion, "version", false, "if true, print version and exit")

	// Allow configuration of envfile path
	// If not set, ParseConfig will not try to load variables to environment from a file
	flag.StringVar(&envFilePath, "envfile", "", "envfile path")

	flag.Parse()

	if printVersion {
		fmt.Printf("v%s build on %s from sha1 %s\n", version, buildTime, sha1ver)
		os.Exit(0)
	}

	opts := &config.ConfigOptions{EnvFilePath: envFilePath}
	cfg, err := config.ParseConfig(opts)
	if err != nil {
		panic(err)
	}

	if err := runServer(cfg); err != nil {
		panic(err)
	}

	os.Exit(0)
}

func runServer(cfg *config.Config) error {
	if cfg == nil {
		return &errors.NilConfigError{}
	}

	// Application wide loggers
	logServer := log.New(os.Stdout, "[SERVER] ", log.LstdFlags|log.Lshortfile)

	logServer.Printf("Starting server (v%s)...\n", version)

	// Flow client
	// TODO: WithInsecure()?
	flowClient, err := client.New(cfg.AccessAPIHost, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer func() {
		if err := flowClient.Close(); err != nil {
			logServer.Println(err)
		}
	}()

	// Database
	db, err := store.NewGormDB(cfg)
	if err != nil {
		return err
	}
	defer store.CloseGormDB(db)

	// Datastore
	store := store.NewGormStore(db)

	// Application
	app := app.New(cfg, store, flowClient)

	// HTTP server
	server := pds_http.NewServer(cfg, logServer, app)

	server.ListenAndServe()

	return nil
}
