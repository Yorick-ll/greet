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

	case PumpInstructionCreate:
		fmt.Println("PumpInstructionCreate", txSig)

	default:
		fmt.Println("Unknown pump instruction", txSig)
		return
	}

}
