package elastic_cache

import (
	"context"
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/aws/aws-sdk-go/aws/credentials"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/olivere/elastic/v7"
	"github.com/patrickmn/go-cache"
	"github.com/sha1sum/aws_signing_client"
	"go.uber.org/zap"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"
)

type Index interface {
	GetClient() *elastic.Client

	InstallMappings()

	AddIndexRequest(index string, entity entity.Entity)
	AddIndexRequests(index string, entity []entity.Entity)
	AddUpdateRequest(index string, entity entity.Entity)
	HasRequest(entity entity.Entity) bool
	AddRequest(index string, entity entity.Entity, reqType RequestType)
	GetEntitiesByIndex(index string) []entity.Entity
	GetRequests() []Request
	GetRequest(id string) *Request
	ClearRequests()

	Save(index string, entity entity.Entity)
	BatchPersist() bool
	Persist() int

	DeleteByID(id string, index string)
	DeleteBlockNumGT(height uint64, indices ...string) error
}

type index struct {
	client *elastic.Client
	cache  *cache.Cache
}

type Request struct {
	Index  string
	Entity entity.Entity
	Type   RequestType
}

type RequestType string

var (
	IndexRequest  RequestType = "index"
	UpdateRequest RequestType = "update"
)

const saveAttempts int = 3

func New() (Index, error) {
	client, err := newClient()
	if err != nil {
		zap.L().With(zap.Error(err)).Fatal("ElasticCache: Failed to create client")
	}

	return index{client, cache.New(5*time.Minute, 10*time.Minute)}, err
}

func newClient() (*elastic.Client, error) {
	opts := []elastic.ClientOptionFunc{
		elastic.SetURL(strings.Join(config.Get().ElasticSearch.Hosts, ",")),
		elastic.SetSniff(config.Get().ElasticSearch.Sniff),
		elastic.SetHealthcheck(config.Get().ElasticSearch.HealthCheck),
	}

	if config.Get().ElasticSearch.Debug {
		opts = append(opts, elastic.SetTraceLog(ElasticLogger{}))
	}

	if config.Get().ElasticSearch.Aws {
		creds := credentials.NewStaticCredentials(config.Get().Aws.AccessKey, config.Get().Aws.SecretKey, config.Get().Aws.Token)
		awsClient, err := aws_signing_client.New(v4.NewSigner(creds), nil, "es", config.Get().Aws.Region)
		if err != nil {
			return nil, err
		}

		opts = append(opts, elastic.SetHttpClient(awsClient))
		opts = append(opts, elastic.SetScheme("https"))
		return elastic.NewClient(opts...)
	}

	if config.Get().ElasticSearch.Username != "" {
		opts = append(opts, elastic.SetBasicAuth(
			config.Get().ElasticSearch.Username,
			config.Get().ElasticSearch.Password,
		))
	}

	return elastic.NewClient(opts...)
}

func (i index) GetClient() *elastic.Client {
	return i.client
}

func (i index) InstallMappings() {
	zap.L().Info("ElasticCache: Install Mappings")

	files, err := ioutil.ReadDir(config.Get().ElasticSearch.MappingDir)
	if err != nil {
		zap.L().With(zap.Error(err)).Fatal("ElasticCache: Elastic mappings directory error")
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		b, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", config.Get().ElasticSearch.MappingDir, f.Name()))
		if err != nil {
			zap.L().With(zap.Error(err)).With(zap.String("file", f.Name())).Fatal("ElasticCache: Elastic mappings file error")
		}

		index := fmt.Sprintf("%s.%s.%s", config.Get().Network, config.Get().Index, f.Name()[0:len(f.Name())-len(filepath.Ext(f.Name()))])
		if err = i.createIndex(index, b); err != nil {
			zap.S().With(zap.Error(err)).Fatalf("ElasticCache: Failed to create index %s", index)
		}
	}
}

func (i index) createIndex(index string, mapping []byte) error {
	ctx := context.Background()
	client := i.client

	exists, err := client.IndexExists(index).Do(ctx)
	if err != nil {
		return err
	}

	if exists && config.Get().Reindex {
		zap.S().Infof("ElasticCache: Deleting index %s", index)
		_, err = client.DeleteIndex(index).Do(ctx)
		if err != nil {
			return err
		}
		exists = false
	}

	if !exists {
		createIndex, err := client.CreateIndex(index).BodyString(string(mapping)).Do(ctx)
		if err != nil {
			return err
		}

		if createIndex.Acknowledged {
			zap.S().Infof("ElasticCache: Created index %s", index)
		}
	}

	return nil
}

func (i index) AddIndexRequest(index string, entity entity.Entity) {
	zap.L().With(zap.String("slug", entity.Slug())).Debug("ElasticCache: AddIndexRequest")

	i.AddRequest(index, entity, IndexRequest)
}

func (i index) AddIndexRequests(index string, entities []entity.Entity) {
	for _, entity := range entities {
		i.AddIndexRequest(index, entity)
	}
}

func (i index) AddUpdateRequest(index string, entity entity.Entity) {
	zap.L().With(zap.String("slug", entity.Slug())).Debug("ElasticCache: AddUpdateRequest")

	i.AddRequest(index, entity, UpdateRequest)
}

func (i index) HasRequest(entity entity.Entity) bool {
	_, found := i.cache.Get(entity.Slug())

	return found
}

func (i index) AddRequest(index string, entity entity.Entity, reqType RequestType) {
	zap.L().With(
		zap.String("index", index),
		zap.String("type", string(reqType)),
		zap.String("slug", entity.Slug())).Debug("ElasticCache: AddRequest")

	if cached, found := i.cache.Get(entity.Slug()); found == true && cached.(Request).Type == IndexRequest {
		zap.L().With(zap.String("slug", entity.Slug())).Debug("ElasticCache: Switch update to index")
		reqType = IndexRequest
	}

	i.cache.Set(entity.Slug(), Request{index, entity, reqType}, cache.DefaultExpiration)
}

func (i index) GetEntitiesByIndex(index string) []entity.Entity {
	entities := make([]entity.Entity, 0)
	for _, req := range i.GetRequests() {
		if req.Index == index {
			entities = append(entities, req.Entity)
		}
	}

	return entities
}

func (i index) GetRequests() []Request {
	requests := make([]Request, 0)

	for _, item := range i.cache.Items() {
		requests = append(requests, item.Object.(Request))
	}

	return requests
}

func (i index) GetRequest(id string) *Request {
	if item, found := i.cache.Get(id); found == true {
		req := item.(Request)
		return &req
	} else {
		return nil
	}
}

func (i index) ClearRequests() {
	i.cache.Flush()
}

func (i index) Save(index string, entity entity.Entity) {
	i.save(index, entity, 1)
}

func (i index) save(index string, entity entity.Entity, attempt int) {
	if attempt > saveAttempts {
		zap.L().With(zap.String("index", index), zap.String("slug", entity.Slug())).
			Fatal("ElasticCache: Failed to save entity, Too many attempts")
	}

	_, err := i.client.Index().
		Index(index).
		Id(entity.Slug()).
		BodyJson(entity).
		Do(context.Background())

	if err != nil {
		zap.L().With(zap.Error(err), zap.String("index", index), zap.String("slug", entity.Slug())).
			Error("ElasticCache: Failed to save entity")
		time.Sleep(1 * time.Second)

		i.save(index, entity, attempt+1)
	}
}

func (i index) BatchPersist() bool {
	if len(i.GetRequests()) < 500 {
		return false
	}

	actions := len(i.GetRequests())
	start := time.Now()
	i.Persist()

	zap.L().With(
		zap.Duration("elapsed", time.Since(start)),
		zap.Int("actions", actions),
	).Info("ElasticCache: Persisting data")

	return true
}

func (i index) Persist() int {
	bulk := i.client.Bulk()
	for _, r := range i.GetRequests() {
		if r.Type == IndexRequest {
			bulk.Add(elastic.NewBulkIndexRequest().Index(r.Index).Id(r.Entity.Slug()).Doc(r.Entity))
		} else if r.Type == UpdateRequest {
			bulk.Add(elastic.NewBulkUpdateRequest().Index(r.Index).Id(r.Entity.Slug()).Doc(r.Entity))
		}

		actions := bulk.NumberOfActions()
		if actions >= config.Get().ElasticSearch.BulkPersistCount {
			i.persist(bulk)
			bulk = i.client.Bulk()
		}
	}

	actions := bulk.NumberOfActions()
	if actions != 0 {
		i.persist(bulk)
	}

	return actions
}

func (i index) persist(bulk *elastic.BulkService) {
	actions := bulk.NumberOfActions()
	zap.S().Debugf("ElasticCache: Persisting %d actions", actions)

	response, err := bulk.Do(context.Background())
	if err != nil {
		if err.Error() == "elastic: Error 429 (Too Many Requests)" {
			zap.L().With(zap.Error(err)).Warn("ElasticCache: 429 (Too Many Requests)")
			time.Sleep(5 * time.Second)
			i.persist(bulk)
			return
		}
		zap.L().With(zap.Error(err)).Fatal("ElasticCache: Failed to persist requests")
	}

	if response.Errors == true {
		for _, failed := range response.Failed() {
			zap.L().With(
				zap.Any("error", failed.Error),
				zap.String("index", failed.Index),
				zap.String("id", failed.Id),
			).Fatal("ElasticCache: Failed to persist requests")
		}
	}

	zap.L().Debug("ElasticCache: Flushing ES cache")
	i.cache.Flush()
}

func (i index) DeleteByID(id string, index string) {
	i.client.Delete().Index(index).Id(id).Do(context.Background())
	i.client.Flush(index)

	zap.S().Infof("Deleted %s from index %s", id, index)
}

func (i index) DeleteBlockNumGT(height uint64, indices ...string) error {
	_, err := i.client.DeleteByQuery(indices...).
		Query(elastic.NewRangeQuery("BlockNum").Gt(height)).
		Do(context.Background())

	if err != nil {
		zap.S().With(zap.Error(err)).Fatalf("Could not rewind to %d", height)
		return err
	}

	i.client.Flush(indices...)

	zap.S().Infof("Deleted height greater than %d", height)

	return nil
}
