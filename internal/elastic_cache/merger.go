package elastic_cache

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"go.uber.org/zap"
)

func mergeRequests(index string, cached Request, action RequestAction, e entity.Entity) entity.Entity {
	switch {
	case index == TransactionIndex.Get():
		return cached.Entity.(entity.Transaction)

	case index == ContractIndex.Get():
		result := cached.Entity.(entity.Contract)
		if action == ContractSetBaseUri {
			result.BaseUri = e.(entity.Contract).BaseUri
		} else {
			result = e.(entity.Contract)
		}
		return result

	case index == NftIndex.Get():
		result := cached.Entity.(entity.NFT)
		if action == Zrc1Transfer {
			result.Owner = e.(entity.NFT).Owner
		}

		if action == Zrc1DuckRegeneration {
			result.TokenUri = e.(entity.NFT).TokenUri
		}

		if action == Zrc1Burn {
			result.BurnedAt = e.(entity.NFT).BurnedAt
		}

		if action == Zrc6SetBaseUri {
			result.TokenUri = e.(entity.NFT).TokenUri
		}

		if action == Zrc6Transfer {
			result.Owner = e.(entity.NFT).Owner
		}

		if action == Zrc6Burn {
			result.BurnedAt = e.(entity.NFT).BurnedAt
		}

		return result
	}

	zap.L().Fatal("Failed to merge request")
	return nil
}