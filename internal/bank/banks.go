package bank

import (
	"github.com/Philanthropists/toshl-email-autosync/internal/bank/bancolombia"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync/types"
)

var banks []types.BankDelegate

func init() {
	banks = []types.BankDelegate{bancolombia.Bancolombia{}}
}

func GetBanks() []types.BankDelegate {
	return banks
}
