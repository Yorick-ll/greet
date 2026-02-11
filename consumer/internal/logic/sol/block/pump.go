package block

import (
	"encoding/binary"
	"fmt"

	"github.com/blocto/solana-go-sdk/client"
	"github.com/blocto/solana-go-sdk/types"
	"github.com/mr-tron/base58"
)

const (
	PumpInstructionBuy    = 0xeaebda01123d0666
	PumpInstructionSell   = 0xad837f01a485e633
	PumpInstructionCreate = 0x77071c0528c81e18
)

func GetPumpInstruction(data []byte) uint64 {
	if len(data) < 8 {
		return 0
	}
	pumpInstruction := binary.LittleEndian.Uint64(data[:8])
	return pumpInstruction
}

func DecodePumpInstruction(instruction *types.CompiledInstruction, tx *client.BlockTransaction) {
	pumpInstruction := GetPumpInstruction(instruction.Data)
	txSig := base58.Encode(tx.Transaction.Signatures[0])
	switch pumpInstruction {

	case PumpInstructionBuy:
		fmt.Println("PumpInstructionBuy", txSig)
	case PumpInstructionSell:
		fmt.Println("PumpInstructionSell", txSig)
		return
	case PumpInstructionCreate:
		fmt.Println("PumpInstructionCreate", txSig)
		return
	default:
		fmt.Println("Unknown pump instruction", txSig)
		return
	}

	accountKeys := tx.AccountKeys

	if len(instruction.Accounts) != 16 && len(accountKeys) != 14 {
		fmt.Println("pump swap account keys is not valid")
		return
	}

	global := accountKeys[instruction.Accounts[0]]

	fmt.Println("global", global)

	if len(instruction.Accounts) == 16 {
		account := binary.LittleEndian.Uint64(instruction.Data[8:16])
		maxSolCost := binary.LittleEndian.Uint64(instruction.Data[16:24])

		fmt.Println("account", account)
		fmt.Println("max_sol_cost", maxSolCost)
	}

}
