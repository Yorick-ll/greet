package block

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
