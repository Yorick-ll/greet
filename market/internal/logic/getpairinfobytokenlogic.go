package logic

import (
	"context"
	"fmt"

	"greet/market/internal/svc"
	"greet/market/market"
	"greet/model/solmodel"
	"greet/pkg/xcode"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPairInfoByTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

const Sol = 100000

func NewGetPairInfoByTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPairInfoByTokenLogic {
	return &GetPairInfoByTokenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetPairInfoByTokenLogic) GetPairInfoByToken(in *market.GetPairInfoByTokenRequest) (*market.GetPairInfoByTokenResponse, error) {
	pair := &solmodel.Pair{}
	var err error
	fmt.Println("chain id is:", in.ChainId)
	if in.ChainId == int64(Sol) || in.ChainId == 100000 {
		pairModel := solmodel.NewPairModel(l.svcCtx.DB)
		//output chain id and token address
		fmt.Println("****************chain id and token address**************", in.ChainId, in.TokenAddress)
		pair, err = pairModel.FindOneByChainIdTokenAddress(l.ctx, in.ChainId, in.TokenAddress)
		if err != nil {
			fmt.Println("******1111111111111*****", in.ChainId, in.TokenAddress)
			// Check if it's a "not found" error
			if errCode, ok := err.(xcode.XCode); ok && errCode.Code() == xcode.NotingFoundError.Code() {
				return nil, fmt.Errorf("pair not found for token: %s on chain: %d", in.TokenAddress, in.ChainId)
			}
			return nil, err
		}
	}

	fmt.Println("****************pair**************", pair.BaseTokenPrice)

	if pair == nil || pair.Address == "" {
		return nil, fmt.Errorf("pairInfo is nil for token: %s", in.TokenAddress)
	}

	return &market.GetPairInfoByTokenResponse{
		ChainId:                in.ChainId,
		Address:                pair.Address,
		Name:                   pair.Name,
		FactoryAddress:         pair.FactoryAddress,
		BaseTokenAddress:       pair.BaseTokenAddress,
		TokenAddress:           pair.TokenAddress,
		BaseTokenSymbol:        pair.BaseTokenSymbol,
		TokenSymbol:            pair.TokenSymbol,
		BaseTokenDecimal:       pair.BaseTokenDecimal,
		TokenDecimal:           pair.TokenDecimal,
		BaseTokenIsNativeToken: pair.BaseTokenIsNativeToken == 1,
		BaseTokenIsToken0:      pair.BaseTokenIsToken0 == 1,
		InitBaseTokenAmount:    pair.InitBaseTokenAmount,
		InitTokenAmount:        pair.InitTokenAmount,
		CurrentBaseTokenAmount: pair.CurrentBaseTokenAmount,
		CurrentTokenAmount:     pair.CurrentTokenAmount,
		Fdv:                    pair.Fdv,
		// 对内显示 不改成fdv
		MktCap:            pair.MktCap,
		BaseTokenPrice:    pair.BaseTokenPrice,
		TokenPrice:        pair.TokenPrice,
		BlockNum:          pair.BlockNum,
		BlockTime:         pair.BlockTime.Unix(),
		HighestTokenPrice: pair.HighestTokenPrice,
		LatestTradeTime:   pair.LatestTradeTime.Unix(),
	}, nil
}
