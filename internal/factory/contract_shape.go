package factory

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"strings"
)

func CreateContractTransition(name string, args ...string) entity.ContractTransition {
	transition := entity.ContractTransition{
		Name:      name,
		Arguments: map[string]string{},
	}
	for _, arg := range args {
		s := strings.Split(arg, ":")
		transition.Arguments[s[0]] = s[1]
	}

	return transition
}

func IsZrc1(c entity.Contract) bool {
	if c.Address == "0xd793f378a925b9f0d3c4b6ee544d31c707899386" {
		// The Bear Market
		return true
	}
	if c.Address == "0x06f70655d4aa5819e711563eb2383655449f24e9" {
		// NFD
		return true
	}
	if c.Address == "0xea4757c8ba3c8063878c35d0b0eb05c7bce892a1" {
		// Unicutes
		return true
	}

	if !c.ImmutableParams.HasParam("contract_owner", "ByStr20") ||
		!c.ImmutableParams.HasParam("name", "String") ||
		!c.ImmutableParams.HasParam("symbol", "String") {
		return false
	}

	if !c.MutableParams.HasParam("minters", "Map ByStr20 Dummy") ||
		!c.MutableParams.HasParam("token_owners", "Map Uint256 ByStr20") ||
		!c.MutableParams.HasParam("owned_token_count", "Map ByStr20 Uint256") ||
		!c.MutableParams.HasParam("token_approvals", "Map Uint256 ByStr20") ||
		!c.MutableParams.HasParam("operator_approvals", "Map ByStr20 (Map ByStr20 Dummy)") ||
		!c.MutableParams.HasParam("token_uris", "Map Uint256 String") ||
		!c.MutableParams.HasParam("total_supply", "Uint256") ||
		!c.MutableParams.HasParam("token_id_count", "Uint256") {
		return false
	}

	if !hasTransition(c, CreateContractTransition("Mint", "to:ByStr20", "token_uri:String")) ||
		!hasTransition(c, CreateContractTransition("Transfer", "to:ByStr20", "token_id:Uint256")) ||
		!hasTransition(c, CreateContractTransition("Burn", "token_id:Uint256")) ||
		!hasTransition(c, CreateContractTransition("TransferFrom", "to:ByStr20", "token_id:Uint256")) {
		return false
	}

	return true
}

func IsZrc2(c entity.Contract) bool {
	if !c.ImmutableParams.HasParam("contract_owner", "ByStr20") ||
		!c.ImmutableParams.HasParam("name", "String") ||
		!c.ImmutableParams.HasParam("symbol", "String") ||
		!c.ImmutableParams.HasParam("decimals", "Uint32") ||
		!c.ImmutableParams.HasParam("init_supply", "Uint128") {
		return false
	}

	if !c.MutableParams.HasParam("total_supply", "Uint128") ||
		!c.MutableParams.HasParam("balances", "Map ByStr20 Uint128") ||
		!c.MutableParams.HasParam("allowances", "Map ByStr20 (Map ByStr20 Uint128)") {
		return false
	}

	if !hasTransition(c, CreateContractTransition("IncreaseAllowance", "spender:ByStr20", "amount:Uint128")) ||
	    !hasTransition(c, CreateContractTransition("DecreaseAllowance", "spender:ByStr20", "amount:Uint128")) ||
		!hasTransition(c, CreateContractTransition("Transfer", "to:ByStr20", "amount:Uint128")) ||
		!hasTransition(c, CreateContractTransition("TransferFrom", "from:ByStr20", "to:ByStr20", "amount:Uint128")) {
		return false
	}

	return true
}

func IsZrc3(c entity.Contract) bool {
	if !c.ImmutableParams.HasParam("contract_owner", "ByStr20") ||
		!c.ImmutableParams.HasParam("name", "String") ||
		!c.ImmutableParams.HasParam("symbol", "String") ||
		!c.ImmutableParams.HasParam("decimals", "Uint32") ||
		!c.ImmutableParams.HasParam("init_supply", "Uint128") {
		return false
	}

	if !c.MutableParams.HasParam("total_supply", "Uint128") ||
		!c.MutableParams.HasParam("balances", "Map ByStr20 Uint128") ||
		!c.MutableParams.HasParam("allowances", "Map ByStr20 (Map ByStr20 Uint128)") ||
		!c.MutableParams.HasParam("void_cheques", "Map ByStr ByStr20") {
		return false
	}

	if !hasTransition(c, CreateContractTransition("IncreaseAllowance", "spender:ByStr20", "amount:Uint128")) ||
	    !hasTransition(c, CreateContractTransition("DecreaseAllowance", "spender:ByStr20", "amount:Uint128")) ||
		!hasTransition(c, CreateContractTransition("Transfer", "to:ByStr20", "amount:Uint128")) ||
		!hasTransition(c, CreateContractTransition("TransferFrom", "from:ByStr20", "to:ByStr20", "amount:Uint128")) ||
		!hasTransition(c, CreateContractTransition("ChequeSend", "pubkey:ByStr20", "to:ByStr20", "amount:Uint128", "fee:Uint128", "nonce:Uint218", "signature:ByStr64")) {
		return false
	}

	return true
}

func IsZrc4(c entity.Contract) bool {
	if !c.ImmutableParams.HasParam("owners_list", "List ByStr20") ||
		!c.ImmutableParams.HasParam("required_signatures", "Uint32") {
		return false
	}

	if !c.MutableParams.HasParam("owners", "Map ByStr20 Bool") ||
		!c.MutableParams.HasParam("transactionCount", "Uint32") ||
		!c.MutableParams.HasParam("signatures", "Map Uint32 (Map ByStr20 Bool)") ||
		!c.MutableParams.HasParam("signature_counts", "Map Uint32 Uint32") ||
		!c.MutableParams.HasParam("transactions", "Map Uint32 Transaction") {
		return false
	}

	if !hasTransition(c, CreateContractTransition("SubmitTransaction", "recipient:ByStr20", "amount:Uint128", "tag:String)")) ||
	    !hasTransition(c, CreateContractTransition("SignTransaction", "transactionId:Uint32")) ||
		!hasTransition(c, CreateContractTransition("ExecuteTransaction", "transactionId:Uint32")) ||
		!hasTransition(c, CreateContractTransition("RevokeSignature", "transactionId:Uint32")) ||
		!hasTransition(c, CreateContractTransition("AddFunds")) {
		return false
	}

	return true
}

func IsZrc6(c entity.Contract) bool {
	if c.Address == "0xd2b54e791930dd7d06ea51f3c2a6cf2c00f165ea" {
		// beanterra
		return true
	}

	if !c.ImmutableParams.HasParam("initial_contract_owner", "ByStr20") ||
		!c.ImmutableParams.HasParam("initial_base_uri", "String") {
		return false
	}

	if !c.MutableParams.HasParam("contract_owner", "ByStr20") ||
		!c.MutableParams.HasParam("base_uri", "String") ||
		!c.MutableParams.HasParam("minters", "Map ByStr20 Bool") ||
		!c.MutableParams.HasParam("token_owners", "Map Uint256 ByStr20") ||
		!c.MutableParams.HasParam("spenders", "Map Uint256 ByStr20") ||
		!c.MutableParams.HasParam("operators", "Map ByStr20 (Map ByStr20 Bool)") ||
		!c.MutableParams.HasParam("token_id_count", "Uint256") ||
		!c.MutableParams.HasParam("balances", "Map ByStr20 Uint256") ||
		!c.MutableParams.HasParam("total_supply", "Uint256") {
		return false
	}

	if	!hasTransition(c, CreateContractTransition("Mint", "to:ByStr20", "token_uri:String")) ||
		!hasTransition(c, CreateContractTransition("AddMinter", "minter:ByStr20")) ||
		!hasTransition(c, CreateContractTransition("RemoveMinter", "minter:ByStr20")) ||
		!hasTransition(c, CreateContractTransition("SetSpender", "spender:ByStr20", "token_id:Uint256")) ||
		!hasTransition(c, CreateContractTransition("AddOperator", "operator:ByStr20")) ||
		!hasTransition(c, CreateContractTransition("RemoveOperator", "operator:ByStr20")) ||
		!hasTransition(c, CreateContractTransition("TransferFrom", "to:ByStr20", "token_id:Uint256")) {
		return false
	}

	return true
}

func hasTransition(c entity.Contract, transition entity.ContractTransition) bool {
	for _, t := range c.Transitions {
		if t.Name != transition.Name {
			continue
		}

		if len(t.Arguments) != len(transition.Arguments) {
			continue
		}

		for key := range transition.Arguments {
			if _, ok := t.Arguments[key]; ok {
				if t.Arguments[key] == transition.Arguments[key] {
					return true
				}
			}
		}
	}

	return false
}