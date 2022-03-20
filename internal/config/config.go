package config

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/log"
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
	MetadataRetries        int
	IpfsHosts              []string
	IpfsTimeout            int

	AssetPath string
	AssetPort string

	Zilliqa       ZilliqaConfig
	ElasticSearch ElasticSearchConfig
	Aws           AwsConfig
}

type AwsConfig struct {
	AccessKey string
	SecretKey string
	Region    string
}

type ZilliqaConfig struct {
	Url     string
	Debug   bool
	Timeout int
}

type ElasticSearchConfig struct {
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

var ipfsHosts = []string{
	"https://gateway.pinata.cloud",
	"https://cloudflare-ipfs.com",
	"https://gateway.ipfs.io",
	"https://ipfs.eth.aragon.network",
}

func Init() {
	err := godotenv.Load(".env")
	if err != nil {
		zap.L().With(zap.Error(err)).Fatal("Unable to init config")
	}

	initLogger()
}
func initLogger() {
	log.NewLogger(Get().Debug)
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
		MetadataRetries:        getInt("METADATA_RETRIES", 3),
		IpfsHosts:              getSlice("IPFS_HOSTS", ipfsHosts, ","),
		IpfsTimeout:            getInt("IPFS_TIMEOUT", 10),
		AssetPath:              getString("ASSET_PATH", "./var/assets"),
		AssetPort:              getString("ASSET_PORT", "8080"),
		Aws: AwsConfig{
			AccessKey: getString("AWS_ACCESS_KEY_ID", ""),
			SecretKey: getString("AWS_SECRET_KEY_ID", ""),
			Region:    getString("AWS_REGION", ""),
		},
		Zilliqa: ZilliqaConfig{
			Url:     getString("ZILLIQA_URL", ""),
			Timeout: getInt("ZILLIQA_TIMEOUT", 30),
			Debug:   getBool("ZILLIQA_DEBUG", false),
		},
		ElasticSearch: ElasticSearchConfig{
			Hosts:            getSlice("ELASTIC_SEARCH_HOSTS", make([]string, 0), ","),
			Sniff:            getBool("ELASTIC_SEARCH_SNIFF", true),
			HealthCheck:      getBool("ELASTIC_SEARCH_HEALTH_CHECK", true),
			Debug:            getBool("ELASTIC_SEARCH_DEBUG", false),
			Username:         getString("ELASTIC_SEARCH_USERNAME", ""),
			Password:         getString("ELASTIC_SEARCH_PASSWORD", ""),
			MappingDir:       getString("ELASTIC_SEARCH_MAPPING_DIR", "/data/mappings"),
			BulkPersistCount: getInt("ELASTIC_SEARCH_BULK_PERSIST_COUNT", 300),
			Refresh:          getString("ELASTIC_SEARCH_REFRESH", "wait_for"),
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
