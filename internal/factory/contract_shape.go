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