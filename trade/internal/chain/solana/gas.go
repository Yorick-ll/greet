package solana

import (
	"context"
	"greet/pkg/sol"

	aSDK "github.com/gagliardetto/solana-go"
	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/zeromicro/go-zero/core/logx"
)

func (tm *TxManager) CreateGasAndJitoByGasFee(ctx context.Context, isAntiMev bool, initiator aSDK.PublicKey, cuLimit uint32, gasFeeInLamport uint64) ([]aSDK.Instruction, uint64, error) {
	var instructionNew aSDK.Instruction
	var instructions []aSDK.Instruction

	jitoFeeInLamport := uint64(0)

	gasPriceMicroLamports := (gasFeeInLamport - sol.GasPerSignature) * 1e6 / uint64(cuLimit)

	var err error
	if gasPriceMicroLamports != 0 {
		instructionNew, err = computebudget.NewSetComputeUnitPriceInstruction(gasPriceMicroLamports).ValidateAndBuild()

		if nil != err {
			return nil, 0, err
		}

		instructions = append(instructions, instructionNew)

		instructionNew, err = computebudget.NewSetComputeUnitLimitInstruction(cuLimit).ValidateAndBuild()
		if nil != err {
			return nil, 0, err
		}

		instructions = append(instructions, instructionNew)
	}

	logx.WithContext(ctx).Debugf("CreateGasAndJitoByGasPrice, initiator=%s, jitoFeeInLamport=%d, gasPrice=%d, cuLimit=%d, isAntiMev=%v",
		initiator, jitoFeeInLamport, gasPriceMicroLamports, cuLimit, isAntiMev)

	feeInLamport := jitoFeeInLamport + gasFeeInLamport
	return instructions, feeInLamport, nil
}
