package svc

import (
	"fmt"
	"greet/consumer/internal/config"
	"greet/model/solmodel"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/blocto/solana-go-sdk/client"
	"github.com/blocto/solana-go-sdk/rpc"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	solclient "github.com/blocto/solana-go-sdk/client"
)

type ServiceContext struct {
	Config         config.Config
	solClientLock  sync.Mutex
	solClientIndex int
	solClient      *solclient.Client
	solClients     []*solclient.Client
	BlockModel     solmodel.BlockModel
	PairModel      solmodel.PairModel
	TokenModel     solmodel.TokenModel
}

func NewServiceContext(c config.Config) *ServiceContext {

	var solClients []*solclient.Client
	fmt.Println("the lenght of config.Cfg.Sol.NodeUrl:", len(config.Cfg.Sol.NodeUrl), "config.Cfg.Sol.NodeUrl:", config.Cfg.Sol.NodeUrl)
	for _, node := range config.Cfg.Sol.NodeUrl {
		client.New(rpc.WithEndpoint(node), rpc.WithHTTPClient(&http.Client{
			Timeout: 10 * time.Second,
		}))
		solClients = append(solClients, client.NewClient(node))
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", c.MySQLConfig.User, c.MySQLConfig.Password, c.MySQLConfig.Host, c.MySQLConfig.Port, c.MySQLConfig.DBName)
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second * 3, // Slow SQL threshold
			LogLevel:                  logger.Warn,     // Log level
			IgnoreRecordNotFoundError: true,            // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      false,           // Don't include params in the SQL log
			Colorful:                  true,
		},
	)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	// websocketClient := websocket.NewWebsocketClient(zrpc.MustNewClient(c.WebsocketService).Conn())

	if err != nil {
		logx.Errorf("connect to mysql error: %v, dsn: %v", err, dsn)
		logx.Must(err)
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxIdleConns(200)
	sqlDB.SetMaxOpenConns(500)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	fmt.Println("the lenght of solClients:", len(solClients))
	return &ServiceContext{
		Config:     c,
		solClients: solClients,
		BlockModel: solmodel.NewBlockModel(db),
	}
}

func (sc *ServiceContext) GetSolClient() *client.Client {
	sc.solClientLock.Lock()
	defer sc.solClientLock.Unlock()
	if len(sc.solClients) == 0 {
		return nil
	}
	sc.solClientIndex++
	index := sc.solClientIndex % len(sc.solClients)
	sc.solClient = sc.solClients[index]
	return sc.solClients[index]
}
