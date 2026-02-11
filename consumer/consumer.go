package main

import (
	"flag"
	"fmt"

	"greet/consumer/consumer"
	"greet/consumer/internal/config"
	"greet/consumer/internal/logic/sol/block"
	"greet/consumer/internal/logic/sol/slot"
	"greet/consumer/internal/server"
	"greet/consumer/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/consumer.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	config.SaveConf(c) // 同步到全局 Cfg，供 config.Cfg.Sol.NodeUrl 等使用
	ctx := svc.NewServiceContext(c)

	sg := service.NewServiceGroup()
	defer sg.Stop()
	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		consumer.RegisterConsumerServer(grpcServer, server.NewConsumerServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})

	sg.Add(s)

	{
		var realChan = make(chan uint64, 50)

		// 消费者
		for i := 0; i < c.Consumer.Concurrency; i++ {
			sg.Add(block.NewBlockService(ctx, "block-real", realChan, i))
		}
		//生产者
		sg.Add(slot.NewSlotServiceGroup(ctx, realChan))
	}

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	sg.Start()
}
