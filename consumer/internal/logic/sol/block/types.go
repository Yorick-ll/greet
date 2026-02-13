package block

import (
	"greet/model/solmodel"
	"time"

	"github.com/blocto/solana-go-sdk/client"
	"github.com/blocto/solana-go-sdk/common"
)

type TokenAccount struct {
	Owner               string // 账户的所有者地址
	TokenAccountAddress string // token account
	TokenAddress        string // token mint
	TokenDecimal        uint8  // 代币的小数位数
	PreValue            int64  // 代币变动之前的余额
	PostValue           int64  // 代币变动之后的余额
	Closed              bool   // 代币账户是否已被关闭
	Init                bool   // 代币是否已经初始化
	PostValueUIString   string // 格式化之后的PostValue
	PreValueUiString    string // 格式化之后的PreValue
}

type DecodedTx struct {
	BlockDb             *solmodel.Block
	SolPrice            float64
	TokenAccountMap     map[string]*TokenAccount
	InnerInstructionMap map[int]*client.InnerInstruction
	Tx                  *client.BlockTransaction
	TxIndex             int
	TxHash              string
	TokenDecimalMap     map[string]uint8
	PumpEvents          []PumpEvent
	PumpEventIndex      int
}

type PumpEvent struct {
	Sign                 uint64
	Mint                 common.PublicKey
	SolAmount            uint64
	TokenAmount          uint64
	IsBuy                bool
	User                 common.PublicKey
	TimeStamp            uint64
	VirtualSolReserves   uint64
	VirtualTokenReserves uint64
}

type TradeWithPair struct {
	Slot                         int64   `json:"slot"`
	ChainId                      string  `json:"chain_id" tag:"true"`     // Tag for chain ID
	ChainIdInt                   int     `json:"chain_id_int" tag:"true"` // Tag for chain ID
	PairAddr                     string  `json:"pair_addr" tag:"true"`    // Tag for address
	TxHash                       string  `json:"tx_hash" tag:"true"`      // Tag for transaction hash, may cause memory overflow; needs periodic roll-up and deletion
	HashId                       string  `json:"hash_id"`
	Maker                        string  `json:"maker"`                             // Address
	Type                         string  `json:"type"`                              // Tag: sell/buy/add_position/remove_position
	BaseTokenAmount              float64 `json:"base_token_amount"`                 // Amount of base token changed
	TokenAmount                  float64 `json:"token_amount"`                      // Amount of non-base token changed
	BaseTokenPriceUSD            float64 `json:"base_token_price_usd"`              // Price of the base token in USD
	TotalUSD                     float64 `json:"total_usd"`                         // Total value in USD
	TokenPriceUSD                float64 `json:"token_price_usd"`                   // Price of the non-base token in USD
	To                           string  `json:"to"`                                // Token recipient address
	BlockNum                     int64   `json:"block_num"`                         // Block height
	BlockTime                    int64   `json:"block_time"`                        // Block time
	TransactionIndex             int     `json:"transaction_index"`                 // Transaction index
	LogIndex                     int     `json:"log_index"`                         // Log index
	SwapName                     string  `json:"swap_name"`                         // Trading pair version
	CurrentTokenInPoolAmount     float64 `json:"current_token_in_pool_amount"`      // Current token amount in pool
	CurrentBaseTokenInPoolAmount float64 `json:"current_base_token_in_pool_amount"` // Current base token amount in pool

	KlineUpDown5m  float64 `json:"kline_up_down_5m"`  // 5-minute price change, used for pushing to websocket
	KlineUpDown1h  float64 `json:"kline_up_down_1h"`  // 1-hour price change, used for pushing to websocket
	KlineUpDown4h  float64 `json:"kline_up_down_4h"`  // 4-hour price change, used for pushing to websocket
	KlineUpDown6h  float64 `json:"kline_up_down_6h"`  // 6-hour price change, used for pushing to websocket
	KlineUpDown24h float64 `json:"kline_up_down_24h"` // 24-hour price change, used for pushing to websocket
	Fdv            float64 `json:"fdv"`               // Market cap, used for pushing to websocket
	Mcap           float64 `json:"mcap"`              // Circulating market cap

	TokenAmountInt     int64 `json:"token_amount_int"` // Not divided by decimal
	BaseTokenAmountInt int64 `json:"base_token_amount_int"`
	Clamp              bool  `json:"clamp"` // true: clamped or in a clamp
	Clipper            bool  `json:"-"`     // true: clamp

	// pump
	PumpPoint                    float64   `json:"pump_point"`    // Pump score
	PumpLaunched                 bool      `json:"pump_launched"` // Pump launched
	PumpMarketCap                float64   `json:"pump_market_cap"`
	PumpOwner                    string    `json:"pump_owner"`
	PumpSwapPairAddr             string    `json:"pump_swap_pair_addr"`
	PumpVirtualBaseTokenReserves float64   `json:"pump_virtual_base_token_reserves,omitempty"`
	PumpVirtualTokenReserves     float64   `json:"pump_virtual_token_reserves,omitempty"`
	PumpStatus                   int       `json:"pump_status"`
	PumpPairAddr                 string    `json:"pump_pair_addr"`
	CreateTime                   time.Time `json:"create_time"`

	// sol
	BaseTokenAccountAddress string `json:"-"`
	TokenAccountAddress     string `json:"-"`
}
