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

		aRegex := regexp.MustCompile("([a-zA-Z_]{1,})[ ]*:[ ]*([a-zA-Z0-9]*)")
		for _, argMatch := range aRegex.FindAllStringSubmatch(transition[3], -1) {
			cTransition.Arguments[argMatch[1]] = argMatch[2]
		}

		transitions = append(transitions, cTransition)
	}

	return
}
