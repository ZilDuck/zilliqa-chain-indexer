package contract

import (
	"github.com/dantudor/zil-indexer/pkg/zil"
)

func isNFT(c zil.Contract) bool {
	if c.AddressBech32 == "zil167flx79fykulp57ykmh9gnf3curcnyux6dcj5e" {
		// The Bear Market
		return true
	}
	if c.AddressBech32 == "zil1qmmsv4w54fvpnec32cltywpk24zf7f8fftmfmp" {
		// NFD
		return true
	}
	if c.AddressBech32 == "zil1afr40j968jqx8puvxhgtp6c9c77w3y4p49a0hw" {
		// Unicutes
		return true
	}

	return hasNftImmutables(c) && hasNftMutables(c) && hasTransitions(c)
}

func hasNftImmutables(c zil.Contract) bool {
	return c.ImmutableParams.HasParam("contract_owner", "ByStr20") &&
		c.ImmutableParams.HasParam("name", "String") &&
		c.ImmutableParams.HasParam("symbol", "String")
}

func hasNftMutables(c zil.Contract) bool {
	return c.MutableParams.HasParam("minters", "Map ByStr20 Dummy") &&
		c.MutableParams.HasParam("token_owners", "Map Uint256 ByStr20") &&
		c.MutableParams.HasParam("owned_token_count", "Map ByStr20 Uint256") &&
		c.MutableParams.HasParam("token_approvals", "Map Uint256 ByStr20") &&
		c.MutableParams.HasParam("operator_approvals", "Map ByStr20 (Map ByStr20 Dummy)") &&
		c.MutableParams.HasParam("token_uris", "Map Uint256 String") &&
		c.MutableParams.HasParam("total_supply", "Uint256") &&
		c.MutableParams.HasParam("token_id_count", "Uint256")
}

func hasTransitions(c zil.Contract) bool {
	return hasTransition(c, "Mint(to:ByStr20,token_uri:String)") &&
		hasTransition(c, "Transfer(to:ByStr20,token_id:Uint256)") &&
		hasTransition(c, "Burn(token_id:Uint256)") &&
		hasTransition(c, "TransferFrom(to:ByStr20,token_id:Uint256)")
}

func hasTransition(c zil.Contract, t string) bool {
	for _, transition := range c.Transitions {
		if transition == t {
			return true
		}
	}
	return false
}
