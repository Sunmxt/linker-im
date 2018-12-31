package gate

import (
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/server/gate/api"
	svcRPC "github.com/Sunmxt/linker-im/server/svc/rpc"
	//"github.com/Sunmxt/linker-im/server/resource"
	"fmt"
	"net/http"
)

var Config *GatewayOptions
var NodeID svcRPC.NodeID

var Handler *http.ServeMux

func registerAPIEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", api.Health)
}

func init() {
	Handler = http.NewServeMux()

	registerAPIEndpoints(Handler)
}

func LogConfigure() {
	log.Infof0("-config=%v", Config.ExternalConfig.String())
	log.Infof0("-log-level=%v", Config.LogLevel.String())
	log.Infof0("-endpoint=%v", Config.APIEndpoint.String())
	log.Infof0("-manage-endpoint=%v", Config.ManageEndpoint.String())
	log.Infof0("-enable-public-management=%v", Config.PublicManagement.String())
	log.Infof0("-redis-endpoint=%v", Config.RedisEndpoint.String())
	log.Infof0("-services-endpoint=\"%v\"", Config.ServiceEndpoints.String())
}

func RegisterResources() error {
	svcEndpointSet := NewServiceEndpointSetFromFlag(Config.ServiceEndpoints)
	svcEndpointSet.GateID = NodeID
	svcEndpointSet.GoKeepalive()
	return nil
}

func Main() {
	fmt.Println("Protocol exporter of Linker IM.")
	config, err := configureParse()
	if config == nil {
		log.Fatalf("%v", err.Error())
		return
	}

	log.Infof0("Linker IM Server Gateway Start.")

	Config = config
	LogConfigure()

	// Log level
	log.Infof0("Log Level is %v.", Config.LogLevel.Value)
	log.SetGlobalLogLevel(Config.LogLevel.Value)

	NodeID = svcRPC.NewNodeID()
	log.Infof0("Gateway Node ID is %v.", NodeID.String())

	// Serve IM API
	log.Infof0("IM API Serve at %v.", config.APIEndpoint.String())
	api_server := http.Server{
		Addr: config.APIEndpoint.String(),
		Handler: log.TagLogHandler(Handler, map[string]interface{}{
			"entity": "http-api",
		}),
	}

	if err = RegisterResources(); err != nil {
		log.Fatalf("Failed to register resource \"%v\".", err.Error())
		return
	}

	log.Trace("APIServer Object:", api_server)
	if err = api_server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to serve API: %s", err.Error())
	}
}
