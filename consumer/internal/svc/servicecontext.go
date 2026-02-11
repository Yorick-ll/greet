package svc

import (
	"fmt"
	"greet/consumer/internal/config"
	"net/http"
	"sync"
	"time"

	"github.com/blocto/solana-go-sdk/client"
	"github.com/blocto/solana-go-sdk/rpc"

	solclient "github.com/blocto/solana-go-sdk/client"
)

type ServiceContext struct {
	Config         config.Config
	solClientLock  sync.Mutex
	solClientIndex int
	solClient      *solclient.Client
	solClients     []*solclient.Client
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

	fmt.Println("the lenght of solClients:", len(solClients))
	return &ServiceContext{
		Config:     c,
		solClients: solClients,
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
