package config

import (
	"log"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
	"github.com/onflow/flow-go-sdk"
)

type Config struct {
	// -- Admin account --

	AdminAddress    string `env:"FLOW_PDS_ADMIN_ADDRESS,notEmpty"`
	AdminKeyIndex   int    `env:"FLOW_PDS_ADMIN_KEY_INDEX" envDefault:"0"`
	AdminKeyType    string `env:"FLOW_PDS_ADMIN_KEY_TYPE" envDefault:"local"`
	AdminPrivateKey string `env:"FLOW_PDS_ADMIN_PRIVATE_KEY,notEmpty"`

	// -- Database --

	DatabaseDSN  string `env:"FLOW_PDS_DATABASE_DSN" envDefault:"pds.db"`
	DatabaseType string `env:"FLOW_PDS_DATABASE_TYPE" envDefault:"sqlite"`

	// -- Host and chain access --

	Host          string       `env:"FLOW_PDS_HOST"`
	Port          int          `env:"FLOW_PDS_PORT" envDefault:"3000"`
	AccessAPIHost string       `env:"FLOW_PDS_ACCESS_API_HOST,notEmpty"`
	ChainID       flow.ChainID `env:"FLOW_PDS_CHAIN_ID" envDefault:"flow-emulator"`

	// -- Google KMS --

	GoogleKMSProjectID  string `env:"FLOW_PDS_GOOGLE_KMS_PROJECT_ID"`
	GoogleKMSLocationID string `env:"FLOW_PDS_GOOGLE_KMS_LOCATION_ID"`
	GoogleKMSKeyRingID  string `env:"FLOW_PDS_GOOGLE_KMS_KEYRING_ID"`

	// -- Misc --

	// Duration for which to wait for a transaction seal, if 0 wait indefinitely.
	// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
	// For more info: https://pkg.go.dev/time#ParseDuration
	TransactionTimeout time.Duration `env:"FLOW_PDS_TRANSACTION_TIMEOUT" envDefault:"0"`

	// Set the starting height for event polling. This won't have any effect if the value in
	// database (chain_event_status[0].latest_height) is greater.
	// If 0 (default) use latest block height if starting fresh (no previous value in database).
	ChainListenerStartingHeight uint64 `env:"FLOW_PDS_EVENTS_STARTING_HEIGHT" envDefault:"0"`
	// Maximum number of blocks to check at once.
	ChainListenerMaxBlocks uint64 `env:"FLOW_PDS_EVENTS_MAX_BLOCKS" envDefault:"100"`
	// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
	// For more info: https://pkg.go.dev/time#ParseDuration
	ChainListenerInterval time.Duration `env:"FLOW_PDS_EVENTS_INTERVAL" envDefault:"10s"`
}

type ConfigOptions struct {
	EnvFilePath string
}

// ParseConfig parses environment variables and flags to a valid Config.
func ParseConfig(opt *ConfigOptions) (*Config, error) {
	if opt != nil && opt.EnvFilePath != "" {
		// Load variables from a file to the environment of the process
		if err := godotenv.Load(opt.EnvFilePath); err != nil {
			log.Printf("Could not load environment variables from file.\n%s\nIf running inside a docker container this can be ignored.\n\n", err)
		}
	}

	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
