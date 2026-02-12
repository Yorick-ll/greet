package config

import (
	"greet/pkg/constants"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
)

var Cfg Config

const SolChainIdInt = 100000

var (
	SolRpcUseFrequency int
)

type Config struct {
	zrpc.RpcServerConf
	Sol         Chain       `json:"Sol,optional"`
	MySQLConfig MySQLConfig `json:"Mysql"`
	Consumer    Consumer    `json:"Consumer,optional"`
}

type Chain struct {
	ChainId    int64    `json:"ChainId"              json:",env=SOL_CHAINID"`
	NodeUrl    []string `json:"NodeUrl"              json:",env=SOL_NODEURL"`
	MEVNodeUrl string   `json:"MevNodeUrl,optional"  json:",env=SOL_MEVNODEURL"`
	WSUrl      string   `json:"WSUrl,optional"       json:",env=SOL_WSURL"`
	StartBlock uint64   `json:"StartBlock,optional"  json:",env=SOL_STARTBLOCK"`
}

type MySQLConfig struct {
	User     string `json:"User"     json:",env=MYSQL_USER"`
	Password string `json:"Password" json:",env=MYSQL_PASSWORD"`
	Host     string `json:"Host"     json:",env=MYSQL_HOST"`
	Port     int    `json:"Port"     json:",env=MYSQL_PORT"`
	DBName   string `json:"DBname"   json:",env=MYSQL_DBNAME"`
}

type Consumer struct {
	Concurrency int `json:"Concurrency" json:",env=CONSUMER_CONCURRENCY"`
}

func SaveConf(cf Config) {
	Cfg = cf
}

func FindChainRpcByChainId(chainId int) (rpc string) {
	var rpcs []string
	var useFrequency *int

	switch chainId {
	case constants.SolChainIdInt:
		rpcs = Cfg.Sol.NodeUrl
		useFrequency = &SolRpcUseFrequency
	default:
		logx.Error("No Rpc Config")
		return
	}

	if len(rpcs) == 0 {
		logx.Error("No Rpc Config")
		return
	}

	*useFrequency++
	index := *useFrequency % len(rpcs)
	rpc = rpcs[index]
	return
}
