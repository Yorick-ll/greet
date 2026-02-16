package block

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"greet/consumer/internal/svc"
	"greet/pkg/constants"
	"strings"

	"github.com/blocto/solana-go-sdk/types"
	"github.com/near/borsh-go"
	"github.com/shopspring/decimal"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	PumpInstructionBuy    = 0xeaebda01123d0666
	PumpInstructionSell   = 0xad837f01a485e633
	PumpInstructionCreate = 0x77071c0528c81e18
)

const TokenReservesDiff = 279900000000000
const SolReservesDiff = 30000000000

const VirtualInitPumpTokenAmount = 1073000191
const InitSolTokenAmount = 0.015

func GetPumpInstruction(data []byte) uint64 {
	if len(data) < 8 {
		return 0
	}
	pumpInstruction := binary.LittleEndian.Uint64(data[:8])
	return pumpInstruction
}

func DecodePumpInstruction(ctx context.Context, sc *svc.ServiceContext, dtx *DecodedTx, instruction *types.CompiledInstruction, index int) (trade *TradeWithPair, err error) {
	// 获取pump instruction
	pumpInstruction := GetPumpInstruction(instruction.Data)
	switch pumpInstruction {
	case PumpInstructionBuy, PumpInstructionSell:
	default:
		return
	}

	tx := dtx.Tx
	tokenAccountMap := dtx.TokenAccountMap
	accountKeys := tx.AccountKeys
	txHash := dtx.TxHash
	solPrice := dtx.SolPrice

	pair := accountKeys[instruction.Accounts[3]].String()
	to := accountKeys[instruction.Accounts[4]].String()
	tokenAccount := accountKeys[instruction.Accounts[5]].String()
	maker := accountKeys[instruction.Accounts[6]].String()

	tokenAccountInfo := tokenAccountMap[tokenAccount]
	if tokenAccountInfo == nil {
		err = fmt.Errorf("tokenaccountinfo not found")
		return
	}

	// 解析Event事件
	dtx.PumpEvents, err = DecodePumpEvent(dtx.Tx.Meta.LogMessages)

	if err != nil {
		logx.Error("pump swap instruction decode event error: %w", err)
	}

	events := dtx.PumpEvents

	if dtx.PumpEventIndex >= len(events) {
		err = fmt.Errorf("pump event not found, i:%v, len:%v, hash:%v", dtx.PumpEventIndex, len(events), txHash)
		logx.Error(err)
		return
	}

	event := events[dtx.PumpEventIndex]
	dtx.PumpEventIndex++
	tokenAddress := event.Mint.String()

	// realTokenReserves := event.VirtualTokenReserves - TokenReservernsDiff

	// realSolReserves := event.VirtualSolReserves - SolReservesDiff

	tokenDecimal := tokenAccountInfo.TokenDecimal
	trade = &TradeWithPair{}

	trade.ChainId = constants.SolChainId
	trade.TxHash = txHash
	trade.PairAddr = pair
	trade.Maker = maker
	trade.To = to
	if event.IsBuy {
		trade.Type = TradeTypeBuy
	} else {
		trade.Type = TradeTypeSell
	}

	trade.BaseTokenAmount = decimal.New(int64(event.SolAmount), -constants.SolDecimal).InexactFloat64()
	trade.TokenAmount = decimal.New(int64(event.TokenAmount), -int32(tokenDecimal)).InexactFloat64()

	trade.BaseTokenPriceUSD = solPrice
	//BaseTokenAmount: 10sol * 130usdt = 1300usdt
	trade.TotalUSD = decimal.NewFromFloat(trade.BaseTokenAmount).Mul(decimal.NewFromFloat(solPrice)).InexactFloat64()
	//total_sol_usd_price / token_amount  = >   1300 / 100token  =  13usd /token
	trade.TokenPriceUSD = decimal.NewFromFloat(trade.TotalUSD).Div(decimal.NewFromFloat(trade.TokenAmount)).InexactFloat64()
	trade.CurrentBaseTokenInPoolAmount = decimal.New(int64(event.VirtualSolReserves), -constants.SolDecimal).InexactFloat64()
	trade.CurrentTokenInPoolAmount = decimal.New(int64(event.VirtualTokenReserves), -int32(tokenDecimal)).InexactFloat64()

	fmt.Println("tarde.TokenPriceUSD:", trade.TokenPriceUSD)
	fmt.Println("tarde.TokenPriceUSD:", trade.PairAddr)
	fmt.Println("tarde.Hash:", trade.TxHash)

	trade.PairInfo = Pair{
		ChainId:                SolChainId,
		Addr:                   pair,
		BaseTokenAddr:          constants.SolBaseTokenAddr,
		BaseTokenDecimal:       9,
		BaseTokenSymbol:        "SOL",
		TokenAddr:              tokenAddress,
		TokenDecimal:           tokenAccountInfo.TokenDecimal,
		BlockTime:              dtx.BlockDb.BlockTime.Unix(),
		BlockNum:               dtx.BlockDb.Slot,
		Name:                   constants.PumpFun,
		InitTokenAmount:        VirtualInitPumpTokenAmount,
		InitBaseTokenAmount:    InitSolTokenAmount,
		CurrentBaseTokenAmount: trade.CurrentBaseTokenInPoolAmount,
		CurrentTokenAmount:     trade.CurrentTokenInPoolAmount,
	}

	return trade, nil
}

func DecodePumpEvent(logs []string) (events []PumpEvent, err error) {
	fmt.Println("length of logs: %d\n", len(logs))

	for i := range logs {
		fmt.Printf("log: %s , index: %d\n", logs[i], i)

		var event PumpEvent

		if len(logs[i]) > 100 && strings.HasPrefix(logs[i], "Program data: vdt/007m") {
			prefix := strings.TrimPrefix(logs[i], "Program data: ")

			// 首先尝试标准bast64编码

			data, err := base64.StdEncoding.DecodeString(prefix)

			if err != nil {
				// 如果标准解码失败，尝试Raw编码
				data, err = base64.RawStdEncoding.DecodeString(prefix)
				if err != nil {
					err = fmt.Errorf("base64 decode error: %w", err)
					return nil, err
				}
			}

			event, err = DeserializePumpEvent(data)

			if err != nil {
				err = fmt.Errorf("deserialize pump event error: %w", err)
				return nil, err
			}

			events = append(events, event)

		}
	}
	return
}

func DeserializePumpEvent(data []byte) (result PumpEvent, err error) {
	err = borsh.Deserialize(&result, data)
	return
}
