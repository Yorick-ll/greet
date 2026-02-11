package main

import (
	"flag"
	"fmt"

	"greet/consumer/consumer"
	"greet/consumer/internal/config"
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
		//生产者
		sg.Add(slot.NewSlotServiceGroup(ctx))
	}

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	sg.Start()
}
