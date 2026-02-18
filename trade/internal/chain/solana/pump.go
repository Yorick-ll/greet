package solana

import (
	"context"
	"errors"
	"fmt"
	pumpfun "greet/pkg/pumpfun/pump"
	"greet/pkg/sol"
	"greet/pkg/xcode"

	ag_solanago "github.com/gagliardetto/solana-go"
	ag_rpc "github.com/gagliardetto/solana-go/rpc"
	"github.com/shopspring/decimal"
	"github.com/zeromicro/go-zero/core/logx"
)

var Decimals2Value = map[uint8]int64{
	0:  1,
	1:  1e1,
	2:  1e2,
	3:  1e3,
	4:  1e4,
	5:  1e5,
	6:  1e6,
	7:  1e7,
	8:  1e8,
	9:  1e9,
	10: 1e10,
	11: 1e11,
	12: 1e12,
	13: 1e13,
	14: 1e14,
	15: 1e15,
	16: 1e16,
	17: 1e17,
	18: 1e18,
}

func (tm *TxManager) CreateMarketOrder4Pumpfun(ctx context.Context, in *CreateMarketTx) ([]ag_solanago.Instruction, error) {
	fmt.Println("**********")

	initiator := in.UserWalletAccount
	// 需要创建一个ata账户，尽管token账户可能存在，当时为了避免网络请求，不做判断，直接认为需要这个费用

	lamportCost := tm.rentFee

	if in.InMint != ag_solanago.WrappedSol && in.OutMint != ag_solanago.WrappedSol {
		return nil, errors.New("ErrWrongMint")

	}
	// 交易方向
	swapDirection := sol.Swap_Direction_Buy

	tokenMint := in.OutMint

	if in.OutMint == ag_solanago.WrappedSol {
		swapDirection = sol.Swap_Direction_Sell
		tokenMint = in.InMint
	}

	logx.WithContext(ctx).Infof("BuyInPumpfun is initlizated by %s", initiator.String())
	// 处理精度
	amtDecimal, err := decimal.NewFromString(in.AmountIn)

	if nil != err {
		return nil, err
	}

	amtDecimal = amtDecimal.Mul(decimal.NewFromInt(Decimals2Value[in.InDecimal]))
	amountUint64 := uint64(amtDecimal.IntPart())

	priceDecimal, err := decimal.NewFromString(in.Price)

	if err != nil {
		return nil, err
	}

	// 1. 计算单元limit和每个单元费用 指令
	instructions, lamportCostFee, err := tm.CreateGasAndJitoByGasFee(ctx, in.IsAntiMev, initiator, sol.PumpFunSwapCU, sol.GasMODE[1])
	lamportCost += lamportCostFee
	if nil != err {
		return nil, err
	}

	// 2. ata账户指令
	instructionNew, err := sol.CreateAtaIdempotent(initiator, initiator, tokenMint, ag_solanago.TokenProgramID)

	if nil != err {
		return nil, err
	}

	instructions = append(instructions, instructionNew)

	// 3. 余额校验
	// 先获取余额
	// buy: amount_in + gas_fee + server_fee
	// sell: gas_fee

	solBalanceInfo, err := tm.Client.GetBalance(ctx, initiator, ag_rpc.CommitmentFinalized)

	if nil != err {
		return nil, err
	}

	solBalance := solBalanceInfo.Value

	serviceFee := uint64(0)
	// 处理买
	if swapDirection == sol.Swap_Direction_Buy {
		if solBalance < uint64(amtDecimal.IntPart()) {
			return nil, xcode.SolBalanceNotEnough
		}

		// 计算服务费，实际购买的sol数量*1%
		serviceFee = uint64(amtDecimal.Mul(sol.ServericeFeePercent).IntPart())
		lamportCost += serviceFee
	}

	// 处理卖
	if lamportCost > solBalance {
		logx.WithContext(ctx).Errorf("Insufficient SOL: need %d lamports, have %d lamports (%.9f SOL needed, %.9f SOL available)",
			lamportCost, solBalance, float64(lamportCost)/1e9, float64(solBalance)/1e9)
		logx.WithContext(ctx).Errorf("Cost breakdown: rentFee=%d, serviceFee=%d, amountUint64=%d, totalCost=%d",
			tm.rentFee, serviceFee, amountUint64, lamportCost)
		return nil, xcode.SolGasNotEnough
	}

	// 指令构建 buy/sell

	if swapDirection == sol.Swap_Direction_Buy {
		logx.WithContext(ctx).Infof("CreateMarketOrder4Pumpfun::Swap_Direction_Buy, amount to buy in sol=%d", amountUint64)

		buyInstruction, err := pumpfun.BuildBuyInstruction(initiator, tokenMint, amountUint64, in.Slippage, tm.Client, priceDecimal.InexactFloat64(), in.InDecimal, in.OutDecimal)
		if nil != err {
			return nil, err
		}
		instructions = append(instructions, buyInstruction)
	} else {
		logx.WithContext(ctx).Infof("CreateMarketOrder4Pumpfun::Swap_Direction_Sell, amount to sell = %d, in.InDecimal=%d", amountUint64, in.InDecimal)
		tokenAta, _, err := ag_solanago.FindAssociatedTokenAddress(initiator, tokenMint)
		if nil != err {
			return nil, err
		}

		buyInstruction, solVolume, err := pumpfun.BuildSellInstruction(tokenAta, initiator, tokenMint, amountUint64, in.Slippage, false, tm.Client, priceDecimal.InexactFloat64(), in.InDecimal, in.OutDecimal)
		if nil != err {
			return nil, err
		}
		instructions = append(instructions, buyInstruction)

		out := decimal.NewFromUint64(solVolume)
		serviceFee = uint64(out.Mul(sol.ServericeFeePercent).IntPart())
	}

	return instructions, nil
}
