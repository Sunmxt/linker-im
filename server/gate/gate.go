package gate

import (
	"fmt"
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/server"
	"github.com/Sunmxt/linker-im/server/dig"
	"github.com/gomodule/redigo/redis"
	gmux "github.com/gorilla/mux"
	"net/http"
)

var Config *GatewayOptions
var NodeID server.NodeID

type Gate struct {
	config *GatewayOptions
	ID     server.NodeID
	HTTP   *http.Server
	Router *gmux.Router
	RPCRouter *gmux.Router
	RPC       *http.Server
	LB  *ServiceLB
	Dig     dig.Registry
	Node    *dig.Node
	Redis *redis.Pool
	Hub *Hub
	fatal chan error
    discover chan *dig.Notification
}

var gate *Gate

func (g *Gate) Run() {
	var err error

	fmt.Println("Protocol exporter of Linker IM.")
	g.config, err = configureParse()
	if g.config == nil {
		log.Fatalf("%v", err.Error())
		return
	}

	log.Infof0("Linker IM Server Gateway Start.")

	// Log level
	log.Infof0("Log Level set to %v.", g.config.LogLevel.Value)
	log.SetGlobalLogLevel(g.config.LogLevel.Value)

	// Node ID
	g.ID = server.NewNodeID()
	log.Infof0("Gateway Node ID is " + g.ID.String() + ".")

	g.fatal = make(chan error)

	// HTTP
	if err = g.InitHTTP(); err != nil {
		log.Fatal("Cannot initialize HTTP: " + err.Error())
		return
	}

	// Core objects.
	if err = g.InitService(); err != nil {
		log.Fatal("Cannot initialize Services: " + err.Error())
		return
	}

	// RPC
	if err = g.InitRPC(); err != nil {
		log.Fatal("Cannot initialize RPC: " + err.Error())
		return
	}

	go g.ServeHTTP()
	go g.ServeRPC()
	go g.Dig()
    go g.DigService()
	go g.Routing()

	if err = <-g.fatal; err != nil {
		log.Fatal(err.Error())
	}

	log.Info0("Exiting...")
}

func Main() {
	gate = &Gate{}
	gate.Run()
}
