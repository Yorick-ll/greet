package svc

import (
	"fmt"
	"greet/market/market"
	"greet/trade/internal/chain/solana"
	"greet/trade/internal/config"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type ServiceContext struct {
	Config        config.Config
	MarketClient  market.MarketClient
	DB            *gorm.DB
	SolTxMananger *solana.TxManager
}

func NewServiceContext(c config.Config) *ServiceContext {

	marketClient := market.NewMarketClient(zrpc.MustNewClient(c.MarketService).Conn())

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		c.MySQLConfig.User,
		c.MySQLConfig.Password,
		c.MySQLConfig.Host,
		c.MySQLConfig.Port,
		c.MySQLConfig.DBName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		panic(fmt.Sprintf("connect to mysql error: %v, dsn: %v", err, dsn))
	}

	inst := &ServiceContext{
		Config:       c,
		DB:           db,
		MarketClient: marketClient,
	}

	// Initialize SolTxMananger if enabled
	if c.SolConfig.Enable {
		logx.Infof("SolConfig enabled, initializing SolTxMananger...")
		solTxMananger, err := solana.NewTxManager(db, c.SolConfig.NodeUrl[0], c.SolConfig.Jito, c.SolConfig.UUID, c.SimulateOnly)
		if err != nil {
			panic(err)
		}
		inst.SolTxMananger = solTxMananger
		logx.Infof("SolTxMananger initialized successfully")
	} else {
		logx.Infof("SolConfig disabled, SolTxMananger not initialized")
	}

	return inst
}
