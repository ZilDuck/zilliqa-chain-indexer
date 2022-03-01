package factory

import (
	"errors"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/zilliqa"
	"go.uber.org/zap"
	"regexp"
)

type ContractFactory interface {
	CreateContractFromTx(tx entity.Transaction) (*entity.Contract, error)
}

type contractFactory struct {
	zilliqa zilliqa.Service
}

func NewContractFactory(zilliqa zilliqa.Service) ContractFactory {
	return contractFactory{zilliqa}
}

func (f contractFactory) CreateContractFromTx(tx entity.Transaction) (*entity.Contract, error) {
	contractName := f.getContractName(tx.Code)

	if tx.ContractAddress == "" || tx.ContractAddress == "0x" {
		zap.L().With(zap.String("txId", tx.ID)).Warn("ContractAddr Missing from Tx")
		return nil, errors.New("missing contract addr")
	}

	contractValues := make([]zilliqa.ContractValue, 0)
	var err error
	if contractValues, err = f.zilliqa.GetSmartContractInit(tx.ContractAddress[2:]); err != nil {
		zap.L().With(zap.Error(err), zap.String("txID", tx.ID), zap.String("contractAddr", tx.ContractAddress)).Warn("GetSmartContractInit")
		return nil, err
	}

	c := &entity.Contract{
		Address:         tx.ContractAddress,
		BlockNum:        tx.BlockNum,
		Code:            tx.Code,
		Data:            tx.Data,
		Name:            contractName,
		ImmutableParams: f.getImmutableParams(contractValues),
		MutableParams:   f.getMutableParams(tx.Code),
		Transitions:     f.getTransitions(tx.Code),
	}

	c.ZRC1 = IsZrc1(*c)

	c.ZRC6 = IsZrc6(*c)
	if c.ZRC6 {
		if initialBaseUri, err := tx.Data.Params.GetParam("initial_base_uri"); err == nil {
			c.BaseUri = initialBaseUri.Value.Primitive.(string)
		}
	}

	return c, nil
}

func (f contractFactory) getContractName(code string) string {
	r := regexp.MustCompile("(?m)^contract ([a-zA-Z0-9_]*)")
	for _, match := range r.FindAllStringSubmatch(code, 1) {
		return match[1]
	}
	return ""
}

func (f contractFactory) getImmutableParams(coreParams []zilliqa.ContractValue) (params entity.Params) {
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

func (f contractFactory) getTransitions(code string) (transitions []entity.ContractTransition) {
	tRegex := regexp.MustCompile("(?m)transition ([a-zA-Z]*)( \\(|\\()([a-zA-Z0-9_:, ]*)\\)")
	for idx, transition := range tRegex.FindAllStringSubmatch(code, -1) {
		if transition[3] == "" {
			continue
		}

		cTransition := entity.ContractTransition{
			Index:     idx,
			Name:      transition[1],
			Arguments: map[string]string{},
		}

		aRegex := regexp.MustCompile("([a-zA-Z]{1,})[ ]*:[ ]*([a-zA-Z0-9]*)")
		for _, argMatch := range aRegex.FindAllStringSubmatch(transition[3], -1) {
			cTransition.Arguments[argMatch[1]] = argMatch[2]
		}

		transitions = append(transitions, cTransition)
	}

	return
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
	return hasZrc1Immutables(c) && hasZrc1Mutables(c) && hasZrc1Transitions(c)
}

func IsZrc2(c entity.Contract) bool {
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
	return hasTransition(c, CreateContractTransition("Mint", "to:ByStr20", "token_uri:String")) &&
		hasTransition(c, CreateContractTransition("Transfer", "to:ByStr20", "token_id:Uint256")) &&
		hasTransition(c, CreateContractTransition("Burn", "token_id:Uint256")) &&
		hasTransition(c, CreateContractTransition("TransferFrom", "to:ByStr20", "token_id:Uint256"))
}

func hasTransition(c entity.Contract, transition entity.ContractTransition) bool {
	for _, t := range c.Transitions {
		if t.Name != transition.Name {
			continue
		}

		if len(t.Arguments) != len(transition.Arguments) {
			return false
		}
		for key := range t.Arguments {
			if _, ok := transition.Arguments[key]; ok {
				if t.Arguments[key] == transition.Arguments[key] {
					return true
				}
			}
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
	return hasTransition(c, CreateContractTransition("Pause")) &&
		hasTransition(c, CreateContractTransition("Mint", "to:ByStr20", "token_uri:String")) &&
		hasTransition(c, CreateContractTransition("AddMinter", "minter:ByStr20")) &&
		hasTransition(c, CreateContractTransition("RemoveMinter", "minter:ByStr20")) &&
		hasTransition(c, CreateContractTransition("SetSpender", "spender:ByStr20", "token_id:Uint256")) &&
		hasTransition(c, CreateContractTransition("AddOperator", "operator:ByStr20")) &&
		hasTransition(c, CreateContractTransition("RemoveOperator", "operator:ByStr20")) &&
		hasTransition(c, CreateContractTransition("TransferFrom", "to:ByStr20", "token_id:Uint256"))
}
