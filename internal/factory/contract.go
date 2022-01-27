package factory

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/zilliqa"
	"github.com/Zilliqa/gozilliqa-sdk/core"
	"go.uber.org/zap"
	"regexp"
	"strings"
)

type ContractFactory interface {
	CreateContractFromTx(tx entity.Transaction) entity.Contract
}

type contractFactory struct {
	zilliqa zilliqa.Service
}

func NewContractFactory(zilliqa zilliqa.Service) ContractFactory {
	return contractFactory{zilliqa}
}

func (f contractFactory) CreateContractFromTx(tx entity.Transaction) entity.Contract {
	contractName := f.getContractName(tx.Code)

	contractValues := make([]core.ContractValue, 0)
	if contractName != "Resolver" {
		var err error
		if contractValues, err = f.zilliqa.GetSmartContractInit(tx.ContractAddress[2:]); err != nil {
			zap.L().With(zap.Error(err), zap.String("txID", tx.ID)).Fatal("GetSmartContractInit")
		}
	}

	c := entity.Contract{
		Address:         tx.ContractAddress,
		AddressBech32:   tx.ContractAddressBech32,
		BlockNum:        tx.BlockNum,
		Code:            tx.Code,
		Data:            tx.Data,
		Name:            contractName,
		ImmutableParams: f.getImmutableParams(contractValues),
		MutableParams:   f.getMutableParams(tx.Code),
		Transitions:     f.getTransitions(tx.Code),
	}

	c.ZRC1 = IsZrc1(c)

	c.ZRC6 = IsZrc6(c)
	if c.ZRC6 {
		if initialBaseUri, err := tx.Data.Params.GetParam("initial_base_uri"); err == nil {
			c.BaseUri = initialBaseUri.Value.Primitive.(string)
		}
	}

	return c
}

func (f contractFactory) getContractName(code string) string {
	r := regexp.MustCompile("(?m)^contract ([a-zA-Z0-9_]*)")
	for _, match := range r.FindAllStringSubmatch(code, 1) {
		return match[1]
	}
	return ""
}

func (f contractFactory) getImmutableParams(coreParams []core.ContractValue) (params entity.Params) {
	if coreParams == nil {
		return
	}

	for _, contractValue := range coreParams {
		params = append(params, entity.Param{
			Type:  contractValue.Type,
			VName: contractValue.VName,
		})
	}
	return
}

func (f contractFactory) getMutableParams(code string) (params entity.Params) {
	r := regexp.MustCompile("(?m)^field ([a-zA-Z0-9_]*)( :|:) ([a-zA-Z0-9][\\(a-zA-Z0-9 ]*[a-zA-Z0-9\\)])")
	for _, field := range r.FindAllStringSubmatch(code, -1) {
		params = append(params, entity.Param{
			VName: field[1],
			Type:  field[3],
		})
	}
	return
}

func (f contractFactory) getTransitions(code string) (transitions []string) {
	r := regexp.MustCompile("(?m)^transition ([a-zA-Z]*\\([a-zA-Z0-9_:, ]*\\))")
	for _, transition := range r.FindAllStringSubmatch(code, -1) {
		transitions = append(transitions, strings.ReplaceAll(transition[1], " ", ""))
	}
	return
}

func IsZrc1(c entity.Contract) bool {
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

	return hasZrc1Immutables(c) && hasZrc1Mutables(c) && hasZrc1Transitions(c)
}

func IsZrc6(c entity.Contract) bool {
	return hasZrc6Immutables(c) && hasZrc6Mutables(c) && hasZrc6Transitions(c)
}

func hasZrc1Immutables(c entity.Contract) bool {
	return c.ImmutableParams.HasParam("contract_owner", "ByStr20") &&
		c.ImmutableParams.HasParam("name", "String") &&
		c.ImmutableParams.HasParam("symbol", "String")
}

func hasZrc1Mutables(c entity.Contract) bool {
	return c.MutableParams.HasParam("minters", "Map ByStr20 Dummy") &&
		c.MutableParams.HasParam("token_owners", "Map Uint256 ByStr20") &&
		c.MutableParams.HasParam("owned_token_count", "Map ByStr20 Uint256") &&
		c.MutableParams.HasParam("token_approvals", "Map Uint256 ByStr20") &&
		c.MutableParams.HasParam("operator_approvals", "Map ByStr20 (Map ByStr20 Dummy)") &&
		c.MutableParams.HasParam("token_uris", "Map Uint256 String") &&
		c.MutableParams.HasParam("total_supply", "Uint256") &&
		c.MutableParams.HasParam("token_id_count", "Uint256")
}

func hasZrc1Transitions(c entity.Contract) bool {
	return hasTransition(c, "Mint(to:ByStr20,token_uri:String)") &&
		hasTransition(c, "Transfer(to:ByStr20,token_id:Uint256)") &&
		hasTransition(c, "Burn(token_id:Uint256)") &&
		hasTransition(c, "TransferFrom(to:ByStr20,token_id:Uint256)")
}

func hasTransition(c entity.Contract, t string) bool {
	for _, transition := range c.Transitions {
		if transition == t {
			return true
		}
	}
	return false
}

func hasZrc6Immutables(c entity.Contract) bool {
	return c.ImmutableParams.HasParam("initial_contract_owner", "ByStr20") &&
		c.ImmutableParams.HasParam("initial_base_uri", "String")
}

func hasZrc6Mutables(c entity.Contract) bool {
	if c.Address == "0xd2b54e791930dd7d06ea51f3c2a6cf2c00f165ea" {
		return true
	}
	return c.MutableParams.HasParam("contract_owner", "ByStr20") &&
		c.MutableParams.HasParam("base_uri", "String") &&
		c.MutableParams.HasParam("minters", "Map ByStr20 Bool") &&
		c.MutableParams.HasParam("token_owners", "Map Uint256 ByStr20") &&
		c.MutableParams.HasParam("spenders", "Map Uint256 ByStr20") &&
		c.MutableParams.HasParam("operators", "Map ByStr20 (Map ByStr20 Bool)") &&
		c.MutableParams.HasParam("token_id_count", "Uint256") &&
		c.MutableParams.HasParam("balances", "Map ByStr20 Uint256") &&
		c.MutableParams.HasParam("total_supply", "Uint256")
}

func hasZrc6Transitions(c entity.Contract) bool {
	return hasTransition(c, "Pause()") &&
		hasTransition(c, "Mint(to:ByStr20,token_uri:String)") &&
		hasTransition(c, "AddMinter(minter:ByStr20)") &&
		hasTransition(c, "RemoveMinter(minter:ByStr20)") &&
		hasTransition(c, "SetSpender(spender:ByStr20,token_id:Uint256)") &&
		hasTransition(c, "AddOperator(operator:ByStr20)") &&
		hasTransition(c, "RemoveOperator(operator:ByStr20)") &&
		hasTransition(c, "TransferFrom(to:ByStr20,token_id:Uint256)")
}
