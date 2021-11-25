package config

import (
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/log"
	"github.com/getsentry/sentry-go"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"math/big"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Env            string
	Network        string
	Index          string
	Debug          bool
	Reindex        bool
	ReindexSize    uint64
	RewindToHeight uint64

	BulkIndex              bool
	BulkTargetHeight       uint64
	BulkIndexSize          uint64
	BulkIndexContractsFrom int
	BulkIndexNftsFrom      int
	FirstBlockNum          uint64
	Subscribe              bool

	SentryDsn string

	Zilliqa       ZilliqaConfig
	ElasticSearch ElasticSearchConfig
	Aws           AwsConfig
}

type AwsConfig struct {
	AccessKey string
	SecretKey string
	Token     string
	Region    string
}

type ZilliqaConfig struct {
	Url string
}

type ElasticSearchConfig struct {
	Aws              bool
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

func Init() {
	args := os.Args[1:]
	if len(args) < 1 {
		panic("specify the environment")
	}

	err := godotenv.Load(fmt.Sprintf("%s.env", args[0]))
	if err != nil {
		zap.L().With(zap.Error(err)).Fatal("Unable to init config")
	}

	initLogger()

	initSentry()
}
func initLogger() {
	log.NewLogger(Get().Debug, Get().SentryDsn)
}

func initSentry() {
	if Get().SentryDsn != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:         Get().SentryDsn,
			Environment: Get().Env,
			Debug:       Get().Debug,
		}); err != nil {
			zap.L().With(zap.Error(err)).Fatal("Sentry init")
		}
	}
}

func Get() *Config {
	return &Config{
		Env:                    getString("ENV", ""),
		Network:                getString("NETWORK", "zilliqa"),
		Index:                  getString("INDEX_NAME", "xxx"),
		Debug:                  getBool("DEBUG", false),
		Reindex:                getBool("REINDEX", false),
		ReindexSize:            getUint64("REINDEX_SIZE", 50),
		RewindToHeight:         getUint64("REWIND_TO_HEIGHT", 0),
		BulkIndex:              getBool("BULK_INDEX", false),
		BulkTargetHeight:       getUint64("BULK_TARGET_HEIGHT", 0),
		BulkIndexSize:          getUint64("BULK_INDEX_SIZE", 100),
		BulkIndexContractsFrom: getInt("BULK_INDEX_CONTRACTS_FROM", -1),
		BulkIndexNftsFrom:      getInt("BULK_INDEX_NFTS_FROM", -1),
		FirstBlockNum:          getUint64("FIRST_BLOCK_NUM", 0),
		Subscribe:              getBool("SUBSCRIBE", true),
		SentryDsn:              getString("SENTRY_DSN", ""),
		Aws: AwsConfig{
			AccessKey: getString("AWS_ACCESS_KEY", ""),
			SecretKey: getString("AWS_SECRET_KEY", ""),
			Token:     getString("AWS_TOKEN", ""),
			Region:    getString("AWS_REGION", ""),
		},
		Zilliqa: ZilliqaConfig{
			Url: getString("ZILLIQA_URL", ""),
		},
		ElasticSearch: ElasticSearchConfig{
			Aws:              getBool("ELASTIC_SEARCH_AWS", true),
			Hosts:            getSlice("ELASTIC_SEARCH_HOSTS", make([]string, 0), ","),
			Sniff:            getBool("ELASTIC_SEARCH_SNIFF", true),
			HealthCheck:      getBool("ELASTIC_SEARCH_HEALTH_CHECK", true),
			Debug:            getBool("ELASTIC_SEARCH_DEBUG", false),
			Username:         getString("ELASTIC_SEARCH_USERNAME", ""),
			Password:         getString("ELASTIC_SEARCH_PASSWORD", ""),
			MappingDir:       getString("ELASTIC_SEARCH_MAPPING_DIR", "/data/mappings"),
			BulkPersistCount: getInt("ELASTIC_SEARCH_BULK_PERSIST_COUNT", 300),
			Refresh:          getString("ELASTIC_SEARCH_REFRESH", "false"),
		},
	}
}

func getString(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultValue
}

func getInt(key string, defaultValue int) int {
	valStr := getString(key, "")
	val, _, err := big.ParseFloat(valStr, 10, 0, big.ToNearestEven)
	if err != nil {
		return defaultValue
	}

	intVal, _ := val.Int64()
	return int(intVal)
}

func getUint(key string, defaultValue uint) uint {
	return uint(getInt(key, int(defaultValue)))
}

func getUint64(key string, defaultValue uint) uint64 {
	return uint64(getInt(key, int(defaultValue)))
}

func getBool(key string, defaultValue bool) bool {
	valStr := getString(key, "")
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}

	return defaultValue
}

func getSlice(key string, defaultVal []string, sep string) []string {
	valStr := getString(key, "")
	if valStr == "" {
		return defaultVal
	}

	return strings.Split(valStr, sep)
}
