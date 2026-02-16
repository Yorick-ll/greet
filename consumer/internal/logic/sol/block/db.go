package block

import (
	"context"
	"errors"
	"fmt"
	"greet/model/solmodel"
	"greet/pkg/constants"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/shopspring/decimal"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

const TradeTypeSell = "sell"
const TradeTypeBuy = "buy"

func (s *BlockService) SaveTrades(ctx context.Context, chainId int64, tradeMap map[string][]*TradeWithPair) {
	group := threading.NewRoutineGroup()

	for key, trade := range tradeMap {
		trade = slice.Filter[*TradeWithPair](trade, func(index int, item *TradeWithPair) bool {
			if item == nil {
				return false
			}
			if item.Type != TradeTypeBuy && item.Type != TradeTypeSell {
				return false
			}
			if item.TokenPriceUSD == 0 {
				return false
			}
			return true
		})

		group.RunSafe(func(key string, trade []*TradeWithPair) func() {
			return func() {
				txhashes := slice.Map[*TradeWithPair, string](trade, func(_ int, item *TradeWithPair) string {
					return item.TxHash
				})
				s.Infof("SaveTrades: will BatchSaveByTrade key: %v, tx hashes: %v", key, txhashes)
				err := s.BatchSaveByTrade(ctx, chainId, key, trade)
				if err != nil {
					s.Errorf("SaveTrades: BatchSaveByTrade err:%v, key:%v, tx hashes: %v", err, key, txhashes)
				}
			}
		}(key, trade))

	}
}

func (s *BlockService) BatchSaveByTrade(ctx context.Context, chainId int64, pairAddress string, trades []*TradeWithPair) (err error) {

	if err = s.SavePairInfo(ctx, chainId, pairAddress, trades); err != nil {
		s.Error(fmt.Errorf("batchSaveByTrade:savePairInfo err:%v", err))
	}
	return
}

func (s *BlockService) SavePairInfo(ctx context.Context, chainId int64, pairAddress string, trades []*TradeWithPair) (err error) {
	fmt.Println("ready to save pair info")

	if trades == nil {
		fmt.Println("trades == nil")

		return
	}
	if len(trades) == 0 {
		fmt.Println("len(trades) == 0")

		return nil
	}
	trade := trades[len(trades)-1]
	var tokenDb *solmodel.Token
	tokenDb, err = s.SaveToken(ctx, trade)

	if err != nil || tokenDb == nil {
		s.Error("SavePairInfo:SaveToken err:", err)
		return err
	}

	if tokenDb.TotalSupply == 0 {
		s.Errorf("savePairInfo token totalSupply is 0, tokenDb: %#v", tokenDb)
	}

	// fill data
	for _, tradeInfo := range trades {
		tradeInfo.PairInfo.TokenTotalSupply = tokenDb.TotalSupply
	}
	_, err = s.SavePair(ctx, trade, tokenDb)

	if err != nil {
		s.Errorf("SavePairInfo:SavePair err: %v", err)
	}

	for _, tradeInfo := range trades {
		tradeInfo.Mcap = trade.Mcap
		tradeInfo.Fdv = trade.Fdv
	}

	fmt.Println("data stored in pair table successfully")

	return
}

func (s *BlockService) SavePair(ctx context.Context, trade *TradeWithPair, tokenDb *solmodel.Token) (pairAtDB *solmodel.Pair, err error) {
	chainId := SolChainIdInt
	var tokenTotalSupply float64
	var tokenSymbol = trade.PairInfo.TokenSymbol
	if tokenDb != nil {
		tokenTotalSupply = tokenDb.TotalSupply
		tokenSymbol = tokenDb.Symbol
	}

	pairAtDB, err = s.sc.PairModel.FindOneByChainIdAddress(ctx, int64(chainId), trade.PairAddr)
	liq := trade.CurrentBaseTokenInPoolAmount*trade.BaseTokenPriceUSD + trade.CurrentTokenInPoolAmount*trade.TokenPriceUSD
	if trade.SwapName == constants.PumpFun {
		liq = trade.CurrentBaseTokenInPoolAmount * trade.BaseTokenPriceUSD * 2
		s.Infof("SavePair: calculated liquidity (PumpFun formula) for pair %v: liq=%v, CurrentBaseTokenInPoolAmount=%v, BaseTokenPriceUSD=%v", trade.PairAddr, liq, trade.CurrentBaseTokenInPoolAmount, trade.BaseTokenPriceUSD)
	} else {
		s.Infof("SavePair: calculated liquidity (standard formula) for pair %v: liq=%v, CurrentBaseTokenInPoolAmount=%v, BaseTokenPriceUSD=%v, CurrentTokenInPoolAmount=%v, TokenPriceUSD=%v", trade.PairAddr, liq, trade.CurrentBaseTokenInPoolAmount, trade.BaseTokenPriceUSD, trade.CurrentTokenInPoolAmount, trade.TokenPriceUSD)
	}

	switch {
	case errors.Is(err, solmodel.ErrNotFound):
		var baseTokenIsNativeToken, baseTokenIsToken0 int64
		if trade.PairInfo.BaseTokenIsNativeToken {
			baseTokenIsNativeToken = 1
		}

		if trade.PairInfo.BaseTokenIsToken0 {
			baseTokenIsToken0 = 1
		}

		//output :SwapName , liq, tokenPrice, base_token_price

		pairAtDB = &solmodel.Pair{
			ChainId: int64(chainId),
			Address: trade.PairAddr,
			Name:    "PumpFun", //因为当前仅测试了Pumpfun，可以先临时设置固定名

			FactoryAddress:               "",
			BaseTokenAddress:             trade.PairInfo.BaseTokenAddr,
			TokenAddress:                 trade.PairInfo.TokenAddr,
			BaseTokenSymbol:              "SOL",
			TokenSymbol:                  tokenSymbol,
			BaseTokenDecimal:             int64(trade.PairInfo.BaseTokenDecimal),
			TokenDecimal:                 int64(trade.PairInfo.TokenDecimal),
			BaseTokenIsNativeToken:       baseTokenIsNativeToken,
			BaseTokenIsToken0:            baseTokenIsToken0,
			CurrentBaseTokenAmount:       trade.CurrentBaseTokenInPoolAmount,
			CurrentTokenAmount:           trade.CurrentTokenInPoolAmount,
			Fdv:                          tokenTotalSupply * trade.TokenPriceUSD,
			MktCap:                       tokenTotalSupply * trade.TokenPriceUSD,
			Liquidity:                    liq,
			TokenPrice:                   trade.TokenPriceUSD,
			BaseTokenPrice:               trade.BaseTokenPriceUSD,
			BlockNum:                     trade.PairInfo.BlockNum,
			BlockTime:                    time.Unix(trade.PairInfo.BlockTime, 0),
			Slot:                         trade.Slot,
			PumpPoint:                    trade.PumpPoint,
			PumpLaunched:                 BoolToInt64(trade.PumpLaunched),
			PumpMarketCap:                trade.PumpMarketCap,
			PumpOwner:                    trade.PumpOwner,
			PumpSwapPairAddr:             trade.PumpSwapPairAddr,
			PumpVirtualBaseTokenReserves: trade.PumpVirtualBaseTokenReserves,
			PumpVirtualTokenReserves:     trade.PumpVirtualTokenReserves,
			PumpStatus:                   int64(trade.PumpStatus),
			PumpPairAddr:                 trade.PumpPairAddr,
			LatestTradeTime:              time.Unix(trade.PairInfo.BlockTime, 0),
		}

		trade.Mcap = pairAtDB.MktCap
		trade.Fdv = pairAtDB.Fdv

		if trade.PairInfo.InitBaseTokenAmount > 0 && trade.PairInfo.InitTokenAmount > 0 {
			pairAtDB.InitBaseTokenAmount = trade.PairInfo.InitBaseTokenAmount
			pairAtDB.InitTokenAmount = trade.PairInfo.InitTokenAmount
		}

		// if err := s.initAmount(pairAtDB); err != nil {
		// 	s.Errorf("SavePair:initAmount insert error: %v, pairAtDB: %#v", err, pairAtDB)
		// }

		err = s.sc.PairModel.Insert(ctx, pairAtDB)
		if err != nil {
			if strings.Contains(err.Error(), "Duplicate entry") {
				// db already exists
				pairAtDB, err = s.sc.PairModel.FindOneByChainIdAddress(ctx, int64(chainId), trade.PairAddr)
				if err != nil {
					return nil, err
				}
				return pairAtDB, nil
			}
			err = fmt.Errorf("PairModel.Insert err:%w", err)
			return
		}
		// TODO: add token cache
		// token.TokenSnapCache.Update(int(100000), pairAtDB.TokenAddress, trade.TokenPriceUSD)
	case err == nil:
	default:
		err = fmt.Errorf("PairModel.FindOneByChainIdAddress err:%w", err)
		return
	}
	// logx.Infof("SavePair:%v db token price: %v, trade token price: %v", trade.PairAddr, pairAtDB.TokenPrice, trade.TokenPriceUSD)

	// 默认值
	trade.Mcap = pairAtDB.MktCap
	trade.Fdv = pairAtDB.Fdv

	if trade.Slot > pairAtDB.Slot {
		// s.Infof("SavePair will UpdatePairDBPoint slot: %v, db slot: %v, hash: %v, pair address: %v", trade.Slot, pairAtDB.Slot, trade.TxHash, trade.PairAddr)

		if pairAtDB.InitBaseTokenAmount == 0 || pairAtDB.InitTokenAmount == 0 {
			if trade.PairInfo.InitBaseTokenAmount > 0 && trade.PairInfo.InitTokenAmount > 0 {
				pairAtDB.InitBaseTokenAmount = trade.PairInfo.InitBaseTokenAmount
				pairAtDB.InitTokenAmount = trade.PairInfo.InitTokenAmount
			}
		}

		// s.initAmount(pairAtDB)

		//因为当前仅测试了Pumpfun，可以先临时设置固定名
		pairAtDB.Name = "PumpFun"
		pairAtDB.TokenSymbol = tokenSymbol
		pairAtDB.Slot = trade.Slot
		pairAtDB.Liquidity = liq
		pairAtDB.BlockNum = trade.PairInfo.BlockNum
		err = UpdatePairDBPoint(trade, pairAtDB, tokenTotalSupply)
		if err != nil {
			err = fmt.Errorf("UpdatePairDBPoint err:%w", err)
			return
		}

		trade.Mcap = pairAtDB.MktCap
		trade.Fdv = pairAtDB.Fdv

		err = s.sc.PairModel.Update(ctx, pairAtDB)
		if err != nil {
			err = fmt.Errorf("PairModel.Update err:%w", err)
			return
		}
	}

	return
}

// UpdatePairDBPoint updates trading information with delayed price updates.
func UpdatePairDBPoint(trade *TradeWithPair, pairDB *solmodel.Pair, tokenTotalSupply float64) error {
	currentTokenInPoolAmount := trade.CurrentTokenInPoolAmount
	currentBaseTokenInPoolAmount := trade.CurrentBaseTokenInPoolAmount
	baseTokenPriceUSD := trade.BaseTokenPriceUSD
	tokenPriceUSD := trade.TokenPriceUSD
	tradeTime := trade.BlockTime

	if pairDB.InitTokenAmount == 0 || pairDB.InitBaseTokenAmount == 0 {
		if trade.PairInfo.InitTokenAmount > 0 && trade.PairInfo.InitBaseTokenAmount > 0 {
			pairDB.InitTokenAmount = trade.PairInfo.InitTokenAmount
			pairDB.InitBaseTokenAmount = trade.PairInfo.InitBaseTokenAmount
			logx.Infof("UpdatePairDBPoint:update init token amount,swapName: %v, %v,%v", trade.SwapName, pairDB.InitTokenAmount, pairDB.InitBaseTokenAmount)
		}
	}

	pairDB.PumpPoint = trade.PumpPoint
	pairDB.PumpStatus = int64(trade.PumpStatus)
	pairDB.PumpVirtualBaseTokenReserves = trade.PumpVirtualBaseTokenReserves
	pairDB.PumpVirtualTokenReserves = trade.PumpVirtualTokenReserves
	// logx.Infof("UpdatePairDBPoint:update token address: %v pump ponit: %v", trade.PairInfo.TokenAddr, pairDB.PumpPoint)

	// Reset token price if base token liquidity is critically low, unless from specific swap types.
	// if trade.SwapName != util.SwapNamePump && currentBaseTokenInPoolAmount > 0 && currentBaseTokenInPoolAmount < 0.01 {
	// 	tokenPriceUSD = 0
	// }

	// Return early if the trade is older than the last update.
	// if tradeTime < pairDB.LatestTradeTime.Unix() {
	// 	return nil
	// }

	// Update token and base token prices only if valid.
	if tokenPriceUSD > 0 {
		pairDB.TokenPrice = tokenPriceUSD
		// logx.Infof("UpdatePairDBPoint %v db price:%v, trade price %v,", pairDB.Address, pairDB.TokenPrice, trade.TokenPriceUSD)
		// if trade.TokenPriceUSD != pairDB.TokenPrice {
		// 	logx.Infof("Diff UpdatePairDBPoint %v db price:%v, trade price %v,", pairDB.Address, pairDB.TokenPrice, trade.TokenPriceUSD)
		// }
	}
	pairDB.BaseTokenPrice = baseTokenPriceUSD

	// Update FDV (fully diluted valuation) based on token supply.
	if tokenTotalSupply > 0 {
		pairDB.Fdv = decimal.NewFromFloat(tokenPriceUSD).Mul(decimal.NewFromFloat(tokenTotalSupply)).InexactFloat64()
		pairDB.MktCap = decimal.NewFromFloat(tokenPriceUSD).Mul(decimal.NewFromFloat(tokenTotalSupply)).InexactFloat64()
	}

	// Update current liquidity only if both amounts are positive.
	if currentBaseTokenInPoolAmount > 0 && currentTokenInPoolAmount > 0 {
		pairDB.CurrentBaseTokenAmount = currentBaseTokenInPoolAmount
		pairDB.CurrentTokenAmount = currentTokenInPoolAmount
	}

	// Update the latest trade time.
	pairDB.LatestTradeTime = time.Unix(tradeTime, 0)

	// Calculate market cap based on the current liquidity and prices.
	if pairDB.Name == constants.PumpFun {
		pairDB.Liquidity = decimal.NewFromFloat(baseTokenPriceUSD).Mul(decimal.NewFromFloat(pairDB.CurrentBaseTokenAmount)).Mul(decimal.NewFromFloat(2)).InexactFloat64()
	} else {
		pairDB.Liquidity = decimal.NewFromFloat(tokenPriceUSD).Mul(decimal.NewFromFloat(pairDB.CurrentTokenAmount)).
			Add(decimal.NewFromFloat(baseTokenPriceUSD).Mul(decimal.NewFromFloat(pairDB.CurrentBaseTokenAmount))).InexactFloat64()
	}

	// pairDB.MktCap = tokenPriceUSD*pairDB.CurrentTokenAmount + baseTokenPriceUSD*pairDB.CurrentBaseTokenAmount

	// TODO: Update pair cache.
	// pair.PairCache.Update(pairDB)
	return nil
}

func BoolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
