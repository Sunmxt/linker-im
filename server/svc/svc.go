package svc

import (
	"fmt"
	ilog "github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/server"
	"github.com/Sunmxt/linker-im/server/dig"
	"github.com/gomodule/redigo/redis"
	"net/http"
	"sync"
)

var service *Service

type Service struct {
	Config    *ServiceOptions
	Model     *Model
	Redis     *redis.Pool
	RPCRouter *http.ServeMux
	RPC       *http.Server
	Node      *dig.Node
	Reg       dig.Registry
	ID        server.NodeID
	Session   server.SessionPool
	Auther    server.Authorizer

	fatal    chan error
	serial   TimeSerializer
	gateNode sync.Map
	gateBuf  sync.Map
}

func (svc *Service) Run() {
	var err error

	fmt.Println("Service node of Linker IM.")
	svc.Config, err = configureParse()
	if svc.Config == nil {
		ilog.Fatalf("%v", err.Error())
		return
	}
	ilog.Info0("Linker IM Service start.")

	ilog.Infof0("Log level: %v", svc.Config.LogLevel.Value)
	ilog.SetGlobalLogLevel(svc.Config.LogLevel.Value)

	svc.ID = server.NewNodeID()
	ilog.Info0("Node ID is " + svc.ID.String())
	svc.fatal = make(chan error)

	if err = svc.InitService(); err != nil {
		ilog.Fatal("Cannot initialize service: " + err.Error())
		return
	}

	if err = svc.InitRPC(); err != nil {
		ilog.Fatal("Cannot initialize service:" + err.Error())
		return
	}

	go svc.ServeRPC()
	go svc.Discover()

	if err = <-svc.fatal; err != nil {
		ilog.Fatal(err.Error())
	}
	ilog.Info0("Exiting...")
}

func Main() {
	service = &Service{}
	service.Run()
}
