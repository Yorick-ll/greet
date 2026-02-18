package solmodel

import (
	"context"
	"fmt"
	"time"

	. "github.com/klen-ygs/gorm-zero/gormc/sql"
	"gorm.io/gorm"
)

// avoid unused err
var _ = InitField
var _ TradeModel = (*customTradeModel)(nil)

type (
	// TradeModel is an interface to be customized, add more methods here,
	// and implement the added methods in customTradeModel.
	TradeModel interface {
		tradeModel
		customTradeLogicModel
	}

	customTradeLogicModel interface {
		WithSession(tx *gorm.DB) TradeModel
		GetNativeTokenPrice(ctx context.Context, chainId int64, searchTime time.Time) (float64, error)
	}

	customTradeModel struct {
		*defaultTradeModel
	}
)

func (c customTradeModel) WithSession(tx *gorm.DB) TradeModel {
	newModel := *c.defaultTradeModel
	c.defaultTradeModel = &newModel
	c.conn = tx
	return c
}

// NewTradeModel returns a model for the database table.
func NewTradeModel(conn *gorm.DB) TradeModel {
	return &customTradeModel{
		defaultTradeModel: newTradeModel(conn),
	}
}

func (m *defaultTradeModel) customCacheKeys(data *Trade) []string {
	if data == nil {
		return []string{}
	}
	return []string{}
}

func (m *defaultTradeModel) GetNativeTokenPrice(ctx context.Context, chainId int64, searchTime time.Time) (float64, error) {
	table := getShardTableNameByTime("trade", searchTime)

	var resp Trade
	// Query the current shard (table)
	err := m.conn.WithContext(ctx).Table(table).Where("chain_id = ? AND block_time > ?", chainId, searchTime).
		Order("block_time ASC").First(&resp).Error

	if err != nil {
		// Return any other error
		return 0, err
	}

	// Return the base token price if a record is found
	return resp.BaseTokenPriceUsd, nil
}

// getShardTableNameByTime generates a table name based on the given timestamp (30-minute interval)
func getShardTableNameByTime(baseTable string, t time.Time) string {
	// Format the table name as: baseTable_<year>_<month>_<day>
	return fmt.Sprintf("%s_%d_%02d_%02d",
		baseTable,
		t.Year(),  // Year
		t.Month(), // Month
		t.Day())   // Day
}
