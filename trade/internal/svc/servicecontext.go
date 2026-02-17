package svc

import (
	"fmt"
	"greet/market/market"
	"greet/trade/internal/config"

	"github.com/zeromicro/go-zero/zrpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type ServiceContext struct {
	Config       config.Config
	MarketClient market.MarketClient
	DB           *gorm.DB
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

	return &ServiceContext{
		Config:       c,
		DB:           db,
		MarketClient: marketClient,
	}
}
