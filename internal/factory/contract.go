package factory

import (
	"errors"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/zilliqa"
	"go.uber.org/zap"
	"regexp"
	"time"
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

	contractValues, err := f.getSmartContractInit(tx, 1, nil)
	if err != nil {
		zap.L().With(
			zap.Error(err),
			zap.String("txID", tx.ID),
			zap.String("contract", tx.ContractAddress),
		).Error("GetSmartContractInit")
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
		Standards:       map[entity.ZrcStandard]bool{},
	}

	c.Standards[entity.ZRC1] = IsZrc1(*c)
	c.Standards[entity.ZRC2] = IsZrc2(*c)
	c.Standards[entity.ZRC3] = IsZrc3(*c)
	c.Standards[entity.ZRC4] = IsZrc4(*c)
	c.Standards[entity.ZRC6] = IsZrc6(*c)

	if c.MatchesStandard(entity.ZRC6) {
		if initialBaseUri, err := tx.Data.Params.GetParam("initial_base_uri"); err == nil {
			c.BaseUri = initialBaseUri.Value.Primitive.(string)
		}
	}

	return c, nil
}

func (f contractFactory) getSmartContractInit(tx entity.Transaction, attempt int, err error) ([]zilliqa.ContractValue, error) {
	zap.L().With(zap.String("contract", tx.ContractAddress)).Debug("Get contract state")
	if attempt >= repository.MaxRetries {
		if err == nil {
			err = errors.New("get contract state unknown error")
		}
		return nil, err
	}

	contractValues := make([]zilliqa.ContractValue, 0)
	if contractValues, err = f.zilliqa.GetSmartContractInit(tx.ContractAddress[2:]); err != nil {
		zap.L().With(zap.Error(err), zap.String("txID", tx.ID), zap.String("contract", tx.ContractAddress)).Warn("GetSmartContractInit")

		if err.Error() == "-5:Address does not exist" {
			time.Sleep(2 * time.Second)
			return f.getSmartContractInit(tx, attempt+1, err)
		}
		return nil, err
	}

	return contractValues, nil
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

	idx := 0
	for _, transition := range tRegex.FindAllStringSubmatch(code, -1) {
		if transition[3] == "" {
			continue
		}

		cTransition := entity.ContractTransition{
			Index:     idx,
			Name:      transition[1],
			Arguments: make([]entity.ContractTransitionArgument, 0),
		}

		aRegex := regexp.MustCompile("([a-zA-Z_]+)[ ]*:[ ]*([a-zA-Z0-9]*)")
		for _, argMatch := range aRegex.FindAllStringSubmatch(transition[3], -1) {
			cTransition.Arguments = append(cTransition.Arguments, entity.ContractTransitionArgument{Key: argMatch[1], Value: argMatch[2]})
		}

		transitions = append(transitions, cTransition)
		idx++
	}

	return
}
