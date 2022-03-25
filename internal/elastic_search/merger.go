package elastic_search

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
	case index == ContractStateIndex.Get():
		result := cached.Entity.(entity.ContractState)
		result.State = e.(entity.ContractState).State
	case index == NftIndex.Get():
		result := cached.Entity.(entity.Nft)
		if action == Zrc1Transfer {
			result.Owner = e.(entity.Nft).Owner
		}

		if action == Zrc1DuckRegeneration {
			result.TokenUri = e.(entity.Nft).TokenUri
		}

		if action == Zrc1Burn {
			result.BurnedAt = e.(entity.Nft).BurnedAt
		}

		if action == Zrc6SetBaseUri {
			result.TokenUri = e.(entity.Nft).TokenUri
		}

		if action == Zrc6SetTokenUri{
			result.TokenUri = e.(entity.Nft).TokenUri
			result.Metadata.Uri = e.(entity.Nft).Metadata.Uri
			result.Metadata.IsIpfs = e.(entity.Nft).Metadata.IsIpfs
			result.Metadata.Status = e.(entity.Nft).Metadata.Status
		}

		if action == Zrc6Transfer {
			result.Owner = e.(entity.Nft).Owner
		}

		if action == Zrc6Burn {
			result.BurnedAt = e.(entity.Nft).BurnedAt
		}

		if action == NftMetadata {
			result.Metadata.Attempts = e.(entity.Nft).Metadata.Attempts
			result.Metadata.Error = e.(entity.Nft).Metadata.Error
			result.Metadata.Properties = e.(entity.Nft).Metadata.Properties
		}

		return result
	}

	zap.L().Fatal("Failed to merge request")
	return nil
}
