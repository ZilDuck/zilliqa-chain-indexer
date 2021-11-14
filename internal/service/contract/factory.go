package contract

import (
	"github.com/Zilliqa/gozilliqa-sdk/core"
	"github.com/dantudor/zil-indexer/internal/service/zilliqa"
	"github.com/dantudor/zil-indexer/pkg/zil"
	"regexp"
	"strings"
)

type Factory interface {
	CreateContractFromTx(tx zil.Transaction) (zil.Contract, error)
}

type factory struct {
	zilliqa zilliqa.Service
}

func NewFactory(zilliqa zilliqa.Service) Factory {
	return factory{zilliqa}
}

func (f factory) CreateContractFromTx(tx zil.Transaction) (zil.Contract, error) {
	contractName := f.getContractName(tx.Code)

	contractValues := make([]core.ContractValue, 0)
	if contractName != "Resolver" {
		contractValues, _ = f.zilliqa.GetSmartContractInit(tx.ContractAddressBech32)
	}

	contract := zil.Contract{
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
	contract.ZRC1 = isNFT(contract)

	return contract, nil
}

func (f factory) getContractName(code string) string {
	r := regexp.MustCompile("(?m)^contract ([a-zA-Z0-9_]*)")
	for _, match := range r.FindAllStringSubmatch(code, 1) {
		return match[1]
	}
	return ""
}

func (f factory) getImmutableParams(coreParams []core.ContractValue) (params zil.Params) {
	if coreParams == nil {
		return
	}

	for _, contractValue := range coreParams {
		params = append(params, zil.Param{
			Type:  contractValue.Type,
			VName: contractValue.VName,
		})
	}
	return
}

func (f factory) getMutableParams(code string) (params zil.Params) {
	r := regexp.MustCompile("(?m)^field ([a-zA-Z0-9_]*)( :|:) ([a-zA-Z0-9][\\(a-zA-Z0-9 ]*[a-zA-Z0-9\\)])")
	for _, field := range r.FindAllStringSubmatch(code, -1) {
		params = append(params, zil.Param{
			VName: field[1],
			Type:  field[3],
		})
	}
	return
}

func (f factory) getTransitions(code string) (transitions []string) {
	r := regexp.MustCompile("(?m)^transition ([a-zA-Z]*\\([a-zA-Z0-9_:, ]*\\))")
	for _, transition := range r.FindAllStringSubmatch(code, -1) {
		transitions = append(transitions, strings.ReplaceAll(transition[1], " ", ""))
	}
	return
}
