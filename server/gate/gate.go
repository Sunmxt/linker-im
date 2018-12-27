package gate

import (
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/server/gate/api"
	"net/http"
)

var Config *GatewayOptions

var Handler *http.ServeMux

func registerAPIEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", api.Health)
}

func init() {
	Handler = http.NewServeMux()

	registerAPIEndpoints(Handler)
}

func DumpConfigure() {
	log.Infof0("--config=%v", Config.ExternalConfig.String())
	log.Infof0("--log-level=%v", Config.LogLevel.String())
	log.Infof0("--endpoint=%v", Config.APIEndpoint.String())
	log.Infof0("--manage-endpoint=%v", Config.ManageEndpoint.String())
	log.Infof0("--enable-public-management=%v", Config.PublicManagement.String())
	log.Infof0("--redis-endpoint=%v", Config.RedisEndpoint.String())
}

func Main() {
	config, err := configureParse()
	if config == nil {
		log.Fatalf("%v", err.Error())
		return
	}

	log.Infof0("Linker IM Server Gateway Start.")

	Config = config
	DumpConfigure()

	// Log level
	log.Infof0("Log Level: %v", Config.LogLevel.Value)
	log.SetGlobalLogLevel(Config.LogLevel.Value)

	// Serve IM API
	log.Infof0("IM API Serve at %v", config.APIEndpoint.String())
	api_server := http.Server{
		Addr: config.APIEndpoint.String(),
		Handler: log.TagLogHandler(Handler, map[string]interface{}{
			"entity": "APIRequest",
		}),
	}

	log.Trace("APIServer Object:", api_server)
	api_server.ListenAndServe()
}
