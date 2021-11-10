package transaction

import (
	"github.com/Zilliqa/gozilliqa-sdk/bech32"
	"go.uber.org/zap"
)

func getBech32Address(address string) string {
	if address == "" {
		return ""
	}
	bech32Address, err := bech32.ToBech32Address(address)
	if err != nil {
		zap.L().With(zap.Error(err), zap.String("address", address)).Error("Failed to create bech32 address")
		return ""
	}
	return bech32Address
}
