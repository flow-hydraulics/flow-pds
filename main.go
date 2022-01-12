package main

import (
	"flag"
	"fmt"

	"os"

	"github.com/flow-hydraulics/flow-pds/service/app"
	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/flow-hydraulics/flow-pds/service/config"
	"github.com/flow-hydraulics/flow-pds/service/http"
	"github.com/flow-hydraulics/flow-pds/service/transactions"
	"github.com/onflow/flow-go-sdk/client"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const version = "0.4.0"

var (
	sha1ver   string // sha1 revision used to build the program
	buildTime string // when the executable was built
)

func init() {
	lvl, ok := os.LookupEnv("FLOW_PDS_LOG_LEVEL")
	if !ok {
		// LOG_LEVEL not set, default to info
		lvl = "info"
	}

	ll, err := log.ParseLevel(lvl)
	if err != nil {
		ll = log.DebugLevel
	}

	log.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	log.SetLevel(ll)
}

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
		return fmt.Errorf("config not provided")
	}

	log.Infof("Starting server (v%s)...", version)

	// Flow client
	// TODO: WithInsecure()?
	maxSize := 1024 * 1024 * 64
	flowClient, err := client.New(cfg.AccessAPIHost, grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxSize)),
	)
	if err != nil {
		return err
	}
	defer func() {
		if err := flowClient.Close(); err != nil {
			log.Error(err)
		}
	}()

	// Database
	db, err := common.NewGormDB(cfg)
	if err != nil {
		return err
	}
	defer common.CloseGormDB(db)

	// Migrate app database
	if err := app.Migrate(db); err != nil {
		return err
	}
	if err := transactions.Migrate(db); err != nil {
		return err
	}

	// Application
	app, err := app.New(cfg, db, flowClient, true)
	if err != nil {
		return err
	}

	defer app.Close()

	// HTTP server
	server := http.NewServer(cfg, app)

	server.ListenAndServe()

	return nil
}
