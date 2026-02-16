package block

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/blocto/solana-go-sdk/client"
	"github.com/blocto/solana-go-sdk/common"
	"github.com/blocto/solana-go-sdk/program/token"
	"github.com/blocto/solana-go-sdk/rpc"
	solTypes "github.com/blocto/solana-go-sdk/types"
)

var ProgramOrca = common.PublicKeyFromString("whirLbMiicVdio4qvUfM5KAg6Ct8VwpYzGff3uctyCc")
var ProgramRaydiumConcentratedLiquidity = common.PublicKeyFromString("CAMMCzo5YL8w4VFF8KVHrK22GGUsp5VTaW7grrKgrWqK")
var ProgramMeteoraDLMM = common.PublicKeyFromString("LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo")
var ProgramPhoneNix = common.PublicKeyFromString("PhoeNiXZ8ByJGLkxNfZRnkUfjvmuYqLR89jjFHGqdXY")

var StableCoinSwapDexes = []common.PublicKey{ProgramOrca, ProgramRaydiumConcentratedLiquidity, ProgramMeteoraDLMM, ProgramPhoneNix}

func GetSolBlockInfoDelay(c *client.Client, ctx context.Context, slot uint64) (resp *client.Block, err error) {
	return GetSolBlockInfo(c, ctx, slot)
}

func GetSolBlockInfo(c *client.Client, ctx context.Context, slot uint64) (resp *client.Block, err error) {
	var count int64
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		resp, err = c.GetBlockWithConfig(ctx, slot, client.GetBlockConfig{
			Commitment:         rpc.CommitmentConfirmed,
			TransactionDetails: rpc.GetBlockConfigTransactionDetailsFull,
		})
		switch {
		case err == nil:
			return
		case strings.Contains(err.Error(), "Block not available for slot"):
			count++
			if count > 10 {
				return
			}
			time.Sleep(time.Second)
		case strings.Contains(err.Error(), "limit"):
			count++
			if count > 10 {
				return
			}
			time.Sleep(time.Second)
		default:
			err = fmt.Errorf("GetBlock err:%w", err)
			return
		}
	}
}

func (s *BlockService) GetBlockSolPrice(ctx context.Context, block *client.Block, tokenAccountMap map[string]*TokenAccount) float64 {
	priceList := make([]float64, 0)
	if tokenAccountMap == nil {
		tokenAccountMap = make(map[string]*TokenAccount)
	}

	for i := range block.Transactions {
		tx := &block.Transactions[i]
		accountKeys := tx.AccountKeys
		innerInstructionMap := GetInnerInstructionMap(tx)
		tokenAccountMap, hasChange := FillTokenAccountMap(tx, tokenAccountMap)
		if !hasChange {
			continue
		}
		for _, instruction := range tx.Transaction.Message.Instructions {
			if in(StableCoinSwapDexes, accountKeys[instruction.ProgramIDIndex]) {
				price := GetBlockSolPriceByTransfer(accountKeys, innerInstructionMap[instruction.ProgramIDIndex], tokenAccountMap)
				if price > 0 {
					priceList = append(priceList, price)
				}
			}
		}
		for _, instructions := range tx.Meta.InnerInstructions {
			for i, instruction := range instructions.Instructions {
				if in(StableCoinSwapDexes, accountKeys[instruction.ProgramIDIndex]) {
					innerInstruction := GetInnerInstructionByInner(instructions.Instructions, i, 2)
					price := GetBlockSolPriceByTransfer(accountKeys, innerInstruction, tokenAccountMap)
					if price > 0 {
						priceList = append(priceList, price)
					}
				}
			}
		}
	}
	price := RemoveMinAndMaxAndCalculateAverage(priceList)

	if price > 0 {
		return price
	}

	if s.solPrice > 0 {
		return s.solPrice
	}

	//兜底策略
	b, err := s.sc.BlockModel.FindOneByNearSlot(s.ctx, int64(block.ParentSlot))
	if err != nil || b == nil {
		// todo: init price
		return 0
	}
	return b.SolPrice
}

func RemoveMinAndMaxAndCalculateAverage(nums []float64) float64 {
	if len(nums) == 0 {
		return 0
	}
	if len(nums) == 1 {
		return nums[0]
	}
	if len(nums) == 2 {
		return (nums[0] + nums[1]) / 2
	}

	minVal, maxVal := math.MaxFloat64, -math.MaxFloat64
	minIndex, maxIndex := -1, -1

	for i, num := range nums {
		if num < minVal {
			minVal = num
			minIndex = i
		}
		if num > maxVal {
			maxVal = num
			maxIndex = i
		}
	}

	var filteredNums []float64
	for i, num := range nums {
		if i != minIndex && i != maxIndex {
			filteredNums = append(filteredNums, num)
		}
	}

	sum := 0.0
	for _, num := range filteredNums {
		sum += num
	}
	average := sum / float64(len(filteredNums))

	return average
}

func GetInnerInstructionByInner(instructions []solTypes.CompiledInstruction, startIndex, innerLen int) *client.InnerInstruction {
	if startIndex+innerLen+1 > len(instructions) {
		return nil
	}
	innerInstruction := &client.InnerInstruction{
		Index: uint64(instructions[startIndex].ProgramIDIndex),
	}
	for i := 0; i < innerLen; i++ {
		innerInstruction.Instructions = append(innerInstruction.Instructions, instructions[startIndex+i+1])
	}
	return innerInstruction
}

func in[T comparable](list []T, a T) bool {
	for i := 0; i < len(list); i++ {
		if list[i] == a {
			return true
		}
	}
	return false
}

func GetBlockSolPriceByTransfer(accountKeys []common.PublicKey, innerInstructions *client.InnerInstruction, tokenAccountMap map[string]*TokenAccount) (solPrice float64) {
	if innerInstructions == nil {
		return
	}

	var transferSOL *token.TransferParam
	var transferUSD *token.TransferParam
	var connect bool
	for j := range innerInstructions.Instructions {
		transfer, err := DecodeTokenTransfer(accountKeys, &innerInstructions.Instructions[j])
		if err != nil {
			transferSOL = nil
			transferUSD = nil
			connect = false
			continue
		}
		from := tokenAccountMap[transfer.From.String()]
		if from == nil {
			transferSOL = nil
			transferUSD = nil
			connect = false
			continue
		}
		to := tokenAccountMap[transfer.To.String()]
		if to == nil {
			transferSOL = nil
			transferUSD = nil
			connect = false
			continue
		}

		if from.TokenAddress == TokenStrWrapSol {
			transferSOL = transfer
			if connect && transferUSD != nil {
				solPrice = float64(transferUSD.Amount) / float64(transferSOL.Amount) * 1000
				if IsSwapTransfer(transferSOL, transferUSD, tokenAccountMap) {
					break
				} else {
					transferUSD = nil
				}
			}
			connect = true
		} else if from.TokenAddress == TokenStrUSDC || from.TokenAddress == TokenStrUSDT {
			transferUSD = transfer
			if connect && transferSOL != nil {
				solPrice = float64(transferUSD.Amount) / float64(transferSOL.Amount) * 1000
				if IsSwapTransfer(transferSOL, transferUSD, tokenAccountMap) {
					break
				} else {
					transferSOL = nil
				}
			}
			connect = true
		} else {
			transferSOL = nil
			transferUSD = nil
			connect = false
		}
	}
	if transferSOL != nil && transferUSD != nil && connect {
		solPrice = float64(transferUSD.Amount) / float64(transferSOL.Amount) * 1000
	} else {
		solPrice = 0
	}
	return
}

func IsSwapTransfer(a, b *token.TransferParam, tokenAccountMap map[string]*TokenAccount) bool {
	if a == nil || b == nil {
		return false
	}
	aFrom := tokenAccountMap[a.From.String()]
	aTo := tokenAccountMap[a.To.String()]
	bFrom := tokenAccountMap[b.From.String()]
	bTo := tokenAccountMap[b.To.String()]
	if aFrom == nil || aTo == nil || bFrom == nil || bTo == nil {
		return false
	}
	if aFrom.Owner == bTo.Owner {
		return true
	}
	if bFrom.Owner == aTo.Owner {
		return true
	}
	return false
}

func DecodeTokenTransfer(accountKeys []common.PublicKey, instruction *solTypes.CompiledInstruction) (transfer *token.TransferParam, err error) {
	transfer = &token.TransferParam{}
	if accountKeys[instruction.ProgramIDIndex].String() == common.Token2022ProgramID.String() {
		if len(instruction.Accounts) < 3 {
			err = errors.New("not enough accounts")
			return
		}
		if len(instruction.Data) < 1 {
			err = errors.New("data len too small")
			return
		}
		if instruction.Data[0] == byte(token.InstructionTransfer) {
			if len(instruction.Data) != 9 {
				err = errors.New("data len not equal 9")
				return
			}
			if len(instruction.Accounts) < 3 {
				err = errors.New("account len too small")
				return
			}
			transfer.From = accountKeys[instruction.Accounts[0]]
			transfer.To = accountKeys[instruction.Accounts[1]]
			transfer.Auth = accountKeys[instruction.Accounts[2]]
			transfer.Amount = binary.LittleEndian.Uint64(instruction.Data[1:])
		} else if instruction.Data[0] == byte(token.InstructionTransferChecked) {
			if len(instruction.Data) < 10 {
				err = errors.New("data len not equal 10")
				return
			}
			if len(instruction.Accounts) < 4 {
				err = errors.New("account len too small")
				return
			}
			transfer.From = accountKeys[instruction.Accounts[0]]
			// mint := accountKeys[instruction.Accounts[1]]
			transfer.To = accountKeys[instruction.Accounts[2]]
			transfer.Auth = accountKeys[instruction.Accounts[3]]
			transfer.Amount = binary.LittleEndian.Uint64(instruction.Data[1:10])
			// decimal := instruction.Data[10]
		} else {
			err = errors.New("not transfer Instruction")
			return
		}
		return transfer, nil
	}
	if accountKeys[instruction.ProgramIDIndex].String() != ProgramStrToken {
		err = errors.New("not token program")
		return
	}
	if len(instruction.Accounts) < 3 {
		err = errors.New("not enough accounts")
		return
	}
	if len(instruction.Data) < 1 {
		err = errors.New("data len to0 small")
		return
	}
	if instruction.Data[0] == byte(token.InstructionTransfer) {
		if len(instruction.Data) != 9 {
			err = errors.New("data len not equal 9")
			return
		}
		if len(instruction.Accounts) < 3 {
			err = errors.New("account len too small")
			return
		}
		transfer.From = accountKeys[instruction.Accounts[0]]
		transfer.To = accountKeys[instruction.Accounts[1]]
		transfer.Auth = accountKeys[instruction.Accounts[2]]
		transfer.Amount = binary.LittleEndian.Uint64(instruction.Data[1:])
	} else if instruction.Data[0] == byte(token.InstructionTransferChecked) {
		if len(instruction.Data) != 10 {
			err = errors.New("data len not equal 10")
			return
		}
		if len(instruction.Accounts) < 4 {
			err = errors.New("account len too small")
			return
		}
		transfer.From = accountKeys[instruction.Accounts[0]]
		// mint := accountKeys[instruction.Accounts[1]]
		transfer.To = accountKeys[instruction.Accounts[2]]
		transfer.Auth = accountKeys[instruction.Accounts[3]]
		transfer.Amount = binary.LittleEndian.Uint64(instruction.Data[1:10])
		// decimal := instruction.Data[10]
	} else {
		err = errors.New("not transfer Instruction")
		return
	}

	return
}

func GetInnerInstructionMap(tx *client.BlockTransaction) map[int]*client.InnerInstruction {
	var innerInstructionMap = make(map[int]*client.InnerInstruction)
	for i := range tx.Meta.InnerInstructions {
		innerInstructionMap[int(tx.Meta.InnerInstructions[i].Index)] = &tx.Meta.InnerInstructions[i]
	}
	return innerInstructionMap
}

func FillTokenAccountMap(tx *client.BlockTransaction, tokenAccountMapIn map[string]*TokenAccount) (tokenAccountMap map[string]*TokenAccount, hasTokenChange bool) {
	if tokenAccountMapIn == nil {
		tokenAccountMapIn = make(map[string]*TokenAccount)
	}
	tokenAccountMap = tokenAccountMapIn
	for _, pre := range tx.Meta.PreTokenBalances {
		var tokenAccount = tx.AccountKeys[pre.AccountIndex].String()
		preVaule, _ := strconv.ParseInt(pre.UITokenAmount.Amount, 10, 64)
		tokenAccountMap[tokenAccount] = &TokenAccount{
			Owner:               pre.Owner,
			TokenAccountAddress: tokenAccount,
			TokenAddress:        pre.Mint,
			TokenDecimal:        pre.UITokenAmount.Decimals,
			PreVaule:            preVaule,
			Closed:              true,
			PreValueUIString:    pre.UITokenAmount.UIAmountString,
		}
	}
	for _, post := range tx.Meta.PostTokenBalances {
		var tokenAccount = tx.AccountKeys[post.AccountIndex].String()
		postVaule, _ := strconv.ParseInt(post.UITokenAmount.Amount, 10, 64)
		if tokenAccountMap[tokenAccount] != nil {
			tokenAccountMap[tokenAccount].Closed = false
			tokenAccountMap[tokenAccount].PostValue = postVaule
			if tokenAccountMap[tokenAccount].PostValue != tokenAccountMap[tokenAccount].PreVaule {
				hasTokenChange = true
			}
		} else {
			hasTokenChange = true
			tokenAccountMap[tokenAccount] = &TokenAccount{
				Owner:               post.Owner,
				TokenAccountAddress: tokenAccount,
				TokenAddress:        post.Mint,
				TokenDecimal:        post.UITokenAmount.Decimals,
				PostValue:           postVaule,
				Init:                true,
				PostVauleUIString:   post.UITokenAmount.UIAmountString,
			}
		}
	}
	for i := range tx.Transaction.Message.Instructions {
		instruction := &tx.Transaction.Message.Instructions[i]
		program := tx.AccountKeys[instruction.ProgramIDIndex].String()
		if program == ProgramStrToken {
			DecodeInitAccountInstruction(tx, tokenAccountMap, instruction)
		}
	}
	for _, instructions := range tx.Meta.InnerInstructions {
		for i := range instructions.Instructions {
			instruction := instructions.Instructions[i]
			program := tx.AccountKeys[instruction.ProgramIDIndex].String()
			if program == ProgramStrToken {
				DecodeInitAccountInstruction(tx, tokenAccountMap, &instruction)
			}
		}
	}
	tokenDecimalMap := make(map[string]uint8)
	for _, v := range tokenAccountMap {
		if v.TokenDecimal != 0 {
			tokenDecimalMap[v.TokenAddress] = v.TokenDecimal
		}
	}
	for _, v := range tokenAccountMap {
		if v.TokenDecimal == 0 {
			v.TokenDecimal = tokenDecimalMap[v.TokenAddress]
		}
	}
	return
}

func DecodeInitAccountInstruction(tx *client.BlockTransaction, tokenAccountMap map[string]*TokenAccount, instruction *solTypes.CompiledInstruction) {
	if len(instruction.Data) == 0 {
		return
	}
	var mint, tokenAccount, owner string
	switch token.Instruction(instruction.Data[0]) {
	case token.InstructionInitializeAccount:
		if len(instruction.Accounts) < 3 {
			return
		}
		tokenAccount = tx.AccountKeys[instruction.Accounts[0]].String()
		mint = tx.AccountKeys[instruction.Accounts[1]].String()
		owner = tx.AccountKeys[instruction.Accounts[2]].String()
	case token.InstructionInitializeAccount2:
		if len(instruction.Accounts) < 2 || len(instruction.Data) < 33 {
			return
		}
		tokenAccount = tx.AccountKeys[instruction.Accounts[0]].String()
		mint = tx.AccountKeys[instruction.Accounts[1]].String()
		owner = common.PublicKeyFromBytes(instruction.Data[1:]).String()
	case token.InstructionInitializeAccount3:
		if len(instruction.Accounts) < 2 || len(instruction.Data) < 33 {
			return
		}
		tokenAccount = tx.AccountKeys[instruction.Accounts[0]].String()
		mint = tx.AccountKeys[instruction.Accounts[1]].String()
		owner = common.PublicKeyFromBytes(instruction.Data[1:]).String()
	default:
		return
	}
	if tokenAccountMap[tokenAccount] != nil && tokenAccountMap[tokenAccount].TokenAddress == mint {
		return
	} else {
		tokenAccountMap[tokenAccount] = &TokenAccount{
			Init:                true,
			Owner:               owner,
			TokenAddress:        mint,
			TokenAccountAddress: tokenAccount,
			TokenDecimal:        0,
			PreVaule:            0,
			PostValue:           0,
		}
	}
}
