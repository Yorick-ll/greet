package logic

import (
	"context"
	"time"

	"greet/market/internal/svc"
	"greet/market/market"
	"greet/model/solmodel"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetNativeTokenPriceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetNativeTokenPriceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetNativeTokenPriceLogic {
	return &GetNativeTokenPriceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetNativeTokenPriceLogic) GetNativeTokenPrice(in *market.GetNativeTokenPriceRequest) (*market.GetNativeTokenPriceResponse, error) {
	resp := &market.GetNativeTokenPriceResponse{
		BaseTokenPriceUsd: 0,
	}

	searchTime, err := time.Parse(time.DateTime, in.SearchTime)
	if err != nil {
		return nil, nil
	}

	tradeModel := solmodel.NewTradeModel(l.svcCtx.DB)
	price, err := tradeModel.GetNativeTokenPrice(l.ctx, in.ChainId, searchTime)
	if err != nil {
		return resp, err
	}
	resp.BaseTokenPriceUsd = price
	return resp, nil
}
