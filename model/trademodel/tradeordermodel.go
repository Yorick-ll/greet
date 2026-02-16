package trademodel

import (
	. "github.com/klen-ygs/gorm-zero/gormc/sql"
	"gorm.io/gorm"
)

// avoid unused err
var _ = InitField
var _ TradeOrderModel = (*customTradeOrderModel)(nil)

type (
	// TradeOrderModel is an interface to be customized, add more methods here,
	// and implement the added methods in customTradeOrderModel.
	TradeOrderModel interface {
		tradeOrderModel
		customTradeOrderLogicModel
	}

	customTradeOrderLogicModel interface {
		WithSession(tx *gorm.DB) TradeOrderModel
	}

	customTradeOrderModel struct {
		*defaultTradeOrderModel
	}
)

func (c customTradeOrderModel) WithSession(tx *gorm.DB) TradeOrderModel {
	newModel := *c.defaultTradeOrderModel
	c.defaultTradeOrderModel = &newModel
	c.conn = tx
	return c
}

// NewTradeOrderModel returns a model for the database table.
func NewTradeOrderModel(conn *gorm.DB) TradeOrderModel {
	return &customTradeOrderModel{
		defaultTradeOrderModel: newTradeOrderModel(conn),
	}
}

func (m *defaultTradeOrderModel) customCacheKeys(data *TradeOrder) []string {
	if data == nil {
		return []string{}
	}
	return []string{}
}
