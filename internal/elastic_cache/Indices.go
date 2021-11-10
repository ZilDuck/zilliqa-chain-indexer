package elastic_cache

import (
	"fmt"
	"github.com/dantudor/zil-indexer/internal/config"
)

type Indices string

var (
	TransactionIndex Indices = "transaction"
	ContractIndex    Indices = "contract"
	NftIndex         Indices = "nft"
	ErrorIndex       Indices = "error"
)

// Sets the network and returns the full string
func (i *Indices) Get() string {
	return fmt.Sprintf("%s.%s.%s", config.Get().Network, config.Get().Index, string(*i))
}

func All() []Indices {
	return []Indices{
		TransactionIndex,
		ContractIndex,
		NftIndex,
	}
}
