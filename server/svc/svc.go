package svc

import (
	"fmt"
	ilog "github.com/Sunmxt/linker-im/log"
    "github.com/gomodule/redigo/redis"
    "github.com/Sunmxt/linker-im/server"
    "net/http"
)

var service *Service

type Service struct {
    Config      *ServiceOptions
    Model       *Model
    Redis       *redis.Pool
    RPCRouter   *http.ServeMux
    RPC         *http.Server
    ID          server.NodeID
    fatal       chan error
}

func (svc *Service) Run() {
    var err error

	fmt.Println("Service node of Linker IM.")
    svc.Config, err =  configureParse()
    if svc.Config == nil {
		ilog.Fatalf("%v", err.Error())
		return
    }
	ilog.Info0("Linker IM Service start.")

	ilog.Infof0("Log level: %v", svc.Config.LogLevel.Value)
	ilog.SetGlobalLogLevel(svc.Config.LogLevel.Value)

    svc.ID = server.NewNodeID()
    ilog.Info0("Node ID is " + svc.ID.String())

    if err = svc.InitService(); err != nil {
        ilog.Fatal("Cannot initialize service: " + err.Error())
        return
    }

    if err = svc.InitRPC(); err != nil {
        ilog.Fatal("Cannot initialize service:" + err.Error())
        return
    }

    go svc.ServeRPC()
}

func Main() {
    service = &Service{}
    service.Run()
}
