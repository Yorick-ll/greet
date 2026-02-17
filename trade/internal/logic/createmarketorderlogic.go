package logic

import (
	"context"

	"greet/market/marketclient"
	"greet/model/trademodel"
	"greet/pkg/constants"
	"greet/pkg/xcode"
	"greet/trade/internal/svc"
	"greet/trade/trade"

	"fmt"

	"github.com/shopspring/decimal"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateMarketOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

const TradeType_Market = 1
const OrderStatus_Proc = 2
const TradeType_OneClick = 3 // 一键买卖

func NewCreateMarketOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateMarketOrderLogic {
	return &CreateMarketOrderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateMarketOrderLogic) CreateMarketOrder(in *trade.CreateMarketOrderRequest) (*trade.CreateMarketOrderResponse, error) {

	// 组装参数
	amountDecimal, err := decimal.NewFromString(in.AmountIn)
	if err != nil {
		return nil, err
	}

	if !amountDecimal.IsPositive() {
		return nil, xcode.AmountErr
	}

	var isAutoSlippage int64 = 0
	var isAntiMev int64 = 0

	pairInfo, err := l.svcCtx.MarketClient.GetPairInfoByToken(l.ctx, &marketclient.GetPairInfoByTokenRequest{
		ChainId:      int64(in.ChainId),
		TokenAddress: in.TokenCa,
	})

	if err != nil {
		return nil, fmt.Errorf("err is %s", err)
	}

	if pairInfo == nil {
		return nil, fmt.Errorf("pair info is nil for token %s", in.TokenCa)
	}

	if pairInfo == nil {
		return nil, fmt.Errorf("%s pairInfo %s fdv is 0", pairInfo.Name, pairInfo.Address)
	}

	capDecimal := decimal.NewFromFloat(pairInfo.Fdv)

	baseTokenPrice := decimal.NewFromFloat(pairInfo.BaseTokenPrice)
	tokenPriceUsdDecimal := decimal.NewFromFloat(pairInfo.TokenPrice)

	if baseTokenPrice.IsZero() {
		l.Errorf("BaseTokenPrice is zero for token %s, pair %s - attempting to fetch current price", in.TokenCa, pairInfo.Address)

		if in.ChainId == constants.SolChainIdInt || in.ChainId == 100000 {
			nativePrice, err := l.svcCtx.MarketClient.GetNativeTokenPrice(l.ctx, &marketclient.GetNativeTokenPriceRequest{
				ChainId: int64(in.ChainId),
			})

			if err == nil && nativePrice.BaseTokenPriceUsd > 0 {
				l.Infof("Retrieved SOL price: %f for token %s", nativePrice.BaseTokenPriceUsd, in.TokenCa)

				baseTokenPrice = decimal.NewFromFloat(nativePrice.BaseTokenPriceUsd)
			} else {
				l.Errorf("Failed to get native token price: %v", err)
				return nil, fmt.Errorf("invalid base token price (zero) for token %s and unable to fetch current price", in.TokenCa)
			}
		} else {
			return nil, fmt.Errorf("invalid base token price (zero) for token %s", in.TokenCa)
		}
	}

	tokenPriceDecimal := tokenPriceUsdDecimal.Div(baseTokenPrice)

	orderValueBase := amountDecimal

	if in.SwapType == trade.SwapType_Sell {
		orderValueBase = amountDecimal.Mul(tokenPriceDecimal)
	}

	order := &trademodel.TradeOrder{
		TradeType:      int64(TradeType_Market),
		ChainId:        int64(in.ChainId),
		TokenCa:        in.TokenCa,
		SwapType:       int64(in.SwapType),
		IsAutoSlippage: isAutoSlippage,
		Slippage:       5000, // 50% slippage
		IsAntiMev:      isAntiMev,
		GasType:        1,
		Status:         int64(OrderStatus_Proc),
		OrderCap:       capDecimal,
		OrderAmount:    amountDecimal,
		OrderPriceBase: tokenPriceDecimal,
		OrderValueBase: orderValueBase,
		OrderBasePrice: baseTokenPrice,
		// 是否翻倍出本 1:是 0:否
		DoubleOut:     BoolToInt64(in.DoubleOut),
		DexName:       pairInfo.Name,
		PairCa:        pairInfo.Address,
		WalletAddress: in.UserWalletAddress,
	}

	if in.IsOneClick {
		order.TradeType = int64(TradeType_OneClick)
	}

	fmt.Print("order is:", order)

	return &trade.CreateMarketOrderResponse{}, nil

}

func BoolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
