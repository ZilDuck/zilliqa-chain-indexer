package config

import (
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/log"
	"github.com/spf13/viper"
	"os"
)

var config Config

type Config struct {
	Env   string
	Debug bool

	Network        string
	Index          string
	Reindex        bool
	ReindexSize    uint64
	RewindToHeight *uint64

	BulkIndex struct {
		Active             bool
		Size               uint64
		IndexContractsFrom *uint64
		IndexNftsFrom      *uint64
	}

	FirstBlockNum   uint64
	Subscribe       bool
	MetadataRetries int
	Ipfs            struct {
		Hosts   []string
		Timeout int
	}
	EventsSupported bool

	AdditionalZrc1           []string
	AdditionalZrc6           []string
	ContractsWithoutMetadata map[string]string

	AssetPort string

	Zilliqa struct {
		Url     string
		Debug   bool
		Timeout int
	}
	ElasticSearch struct {
		Hosts            []string
		Sniff            bool
		HealthCheck      bool
		Debug            bool
		Username         string
		Password         string
		MappingDir       string
		BulkPersistCount int
		Refresh          string
	}
	Aws struct {
		AccessKey string
		SecretKey string
		Region    string
	}
}

func Init() {
	viper.SetConfigName("env.yaml")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil { // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}

	if err := viper.Unmarshal(&config); err != nil {
		panic("Failed to unmarshal config")
	}

	_ = os.Setenv("AWS_ACCESS_KEY_ID", config.Aws.AccessKey)
	_ = os.Setenv("AWS_SECRET_KEY_ID", config.Aws.SecretKey)
	_ = os.Setenv("AWS_REGION", config.Aws.Region)

	log.NewLogger(config.Debug)
}

func Get() Config {
	return config
}
