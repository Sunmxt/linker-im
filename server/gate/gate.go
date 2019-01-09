package gate

import (
	"fmt"
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/server"
	"github.com/Sunmxt/linker-im/server/resource"
	"net/http"
)

var Config *GatewayOptions
var NodeID server.NodeID

func LogConfigure() {
	log.Infof0("-config=%v", Config.ExternalConfig.String())
	log.Infof0("-log-level=%v", Config.LogLevel.String())
	log.Infof0("-endpoint=%v", Config.APIEndpoint.String())
	log.Infof0("-manage-endpoint=%v", Config.ManageEndpoint.String())
	log.Infof0("-redis-endpoint=%v", Config.RedisEndpoint.String())
	log.Infof0("-services-endpoint=\"%v\"", Config.ServiceEndpoints.String())
	log.Infof0("-keepalive-period=%v", Config.KeepalivePeriod.String())
}

func RegisterResources() error {
	svcEndpointSet := NewServiceEndpointSetFromFlag(Config.ServiceEndpoints)
	log.Infof0("Register resource \"svc-endpoint\".")
	if err := resource.Registry.Register("svc-endpoint", svcEndpointSet); err != nil {
		return err
	}
	svcEndpointSet.GoKeepalive(NodeID, Config.KeepalivePeriod.Value)
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
	log.Infof0("Log Level set to %v.", Config.LogLevel.Value)
	log.SetGlobalLogLevel(Config.LogLevel.Value)

	NodeID = server.NewNodeID()
	log.Infof0("Gateway Node ID is %v.", NodeID.String())

	// Serve IM API
	httpMux, err := NewHTTPAPIMux()
	log.Infof0("HTTP API Serve at %v.", config.APIEndpoint.String())
	api_server := http.Server{
		Addr: config.APIEndpoint.String(),
		Handler: log.TagLogHandler(httpMux, map[string]interface{}{
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
