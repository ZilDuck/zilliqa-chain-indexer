package elastic_cache

import (
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
)

type Indices string

var (
	TransactionIndex Indices = "transaction"
	ContractIndex    Indices = "contract"
	NftIndex         Indices = "nft"
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
