package solmodel

import (
	"context"
	"greet/pkg/xcode"

	. "github.com/klen-ygs/gorm-zero/gormc/sql"
	"gorm.io/gorm"
)

// avoid unused err
var _ = InitField
var _ PairModel = (*customPairModel)(nil)

type (
	// PairModel is an interface to be customized, add more methods here,
	// and implement the added methods in customPairModel.
	PairModel interface {
		pairModel
		customPairLogicModel
	}

	customPairLogicModel interface {
		WithSession(tx *gorm.DB) PairModel
		FindOneByChainIdTokenAddress(ctx context.Context, chainId int64, tokenAddress string) (*Pair, error)
	}

	customPairModel struct {
		*defaultPairModel
	}
)

func (c customPairModel) WithSession(tx *gorm.DB) PairModel {
	newModel := *c.defaultPairModel
	c.defaultPairModel = &newModel
	c.conn = tx
	return c
}

// NewPairModel returns a model for the database table.
func NewPairModel(conn *gorm.DB) PairModel {
	return &customPairModel{
		defaultPairModel: newPairModel(conn),
	}
}

func (m *defaultPairModel) customCacheKeys(data *Pair) []string {
	if data == nil {
		return []string{}
	}
	return []string{}
}

func (m customPairModel) FindOneByChainIdTokenAddress(ctx context.Context, chainId int64, tokenAddress string) (*Pair, error) {
	var res []Pair
	err := m.conn.WithContext(ctx).Model(&Pair{}).Where("chain_id = ? and token_address = ?", chainId, tokenAddress).Find(&res).Error
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, xcode.NotingFoundError
	}

	var temporaryRes []Pair
	var contractResult Pair
	if len(res) > 0 {
		// Map to store pairs by category
		categoryMap := make(map[string][]Pair)
		// Group pairs by their name (category)
		for _, pair := range res {
			if pair.Name != "" {
				categoryMap[pair.Name] = append(categoryMap[pair.Name], pair)
			}
		}
		// Process categories (RaydiumV4, RaydiumConcentratedLiquidity)
		for _, pairs := range categoryMap {
			var LiquidityPair Pair
			var maxLiquidityCap float64

			// Find the pair with the largest market cap within each category
			for _, pair := range pairs {
				if pair.Liquidity > maxLiquidityCap {
					maxLiquidityCap = pair.Liquidity
					LiquidityPair = pair
				}
			}
			temporaryRes = append(temporaryRes, LiquidityPair)
		}
	}

	res = temporaryRes
	if len(res) == 1 {
		contractResult = res[0]
	} else if len(res) > 1 {
		//Get the pair with the highest market cap
		var maxLiquidityPair Pair
		var maxLiquidityCap float64
		for _, pair := range res {
			if pair.Name == "PumpFun" {
				continue
			}
			if pair.Liquidity > maxLiquidityCap {
				maxLiquidityCap = pair.Liquidity
				maxLiquidityPair = pair
			}
		}
		contractResult = maxLiquidityPair
	}

	return &contractResult, nil
}
