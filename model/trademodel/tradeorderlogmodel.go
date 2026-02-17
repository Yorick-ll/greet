package trademodel

import (
	. "github.com/klen-ygs/gorm-zero/gormc/sql"
	"gorm.io/gorm"
)

// avoid unused err
var _ = InitField
var _ TradeOrderLogModel = (*customTradeOrderLogModel)(nil)

type (
	// TradeOrderLogModel is an interface to be customized, add more methods here,
	// and implement the added methods in customTradeOrderLogModel.
	TradeOrderLogModel interface {
		tradeOrderLogModel
		customTradeOrderLogLogicModel
	}

	customTradeOrderLogLogicModel interface {
		WithSession(tx *gorm.DB) TradeOrderLogModel
	}

	customTradeOrderLogModel struct {
		*defaultTradeOrderLogModel
	}
)

func (c customTradeOrderLogModel) WithSession(tx *gorm.DB) TradeOrderLogModel {
	newModel := *c.defaultTradeOrderLogModel
	c.defaultTradeOrderLogModel = &newModel
	c.conn = tx
	return c
}

// NewTradeOrderLogModel returns a model for the database table.
func NewTradeOrderLogModel(conn *gorm.DB) TradeOrderLogModel {
	return &customTradeOrderLogModel{
		defaultTradeOrderLogModel: newTradeOrderLogModel(conn),
	}
}

func (m *defaultTradeOrderLogModel) customCacheKeys(data *TradeOrderLog) []string {
	if data == nil {
		return []string{}
	}
	return []string{}
}
