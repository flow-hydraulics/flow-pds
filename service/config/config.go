package config

import (
	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	// -- Admin (or the PDS) account --

	AdminAddress    string `env:"FLOW_PDS_ADMIN_ADDRESS,notEmpty"`
	AdminPrivateKey string `env:"FLOW_PDS_ADMIN_PRIVATE_KEY,notEmpty"`
	// TODO AdminPrivateKeyIndexes

	// -- Database --

	DatabaseDSN  string `env:"FLOW_PDS_DATABASE_DSN" envDefault:"pds.db"`
	DatabaseType string `env:"FLOW_PDS_DATABASE_TYPE" envDefault:"sqlite"`

	// -- Host and chain access --

	Host          string `env:"FLOW_PDS_HOST"`
	Port          int    `env:"FLOW_PDS_PORT" envDefault:"3000"`
	AccessAPIHost string `env:"FLOW_PDS_ACCESS_API_HOST" envDefault:"localhost:3569"`

	// -- Rates etc. ---

	// How many transactions to send per second at max
	SendTransactionRate int `env:"FLOW_PDS_SEND_RATE" envDefault:"10"`

	// -- Testing --

	TestNOCollectibles int `env:"TEST_COLLECTIBLES" envDefault:"20"`
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
