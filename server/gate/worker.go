package gate

import (
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/server/dig"
	"strings"
	"time"
)

func discover(reg dig.Registry) {
	var (
		svc dig.Service
		err error
	)
	for {
		svc, err = reg.Service("linker-gateway")
		if err != nil {
			log.Error("Cannot get service \"linker-gateway\":")
		} else if svc == nil {
			log.Error("Nil value of service \"linker-gateway\".")
		} else {
			break
		}
		time.Sleep(time.Second)
	}

	svc.Publish(&dig.Node{
		Name: "gateway-" + NodeID.String(),
		Metadata: map[string]string{
			"linker-gateway-rpc":    Config.RPCPublishEndpoint.String(),
			"linker-gateway-nodeid": NodeID.String(),
		},
		Timeout: 300,
	})

	for {
		changed, err := reg.Poll()
		if err != nil {
			log.Error("Dig polling failure: " + err.Error())
		}
		if changed {
			log.Info2("Dig state changed.")
			log.Info0("Nodes of service \"linker-gateway\": " + strings.Join(svc.Nodes(), ",") + ".")
		}
		log.DebugLazy(func() string {
			return "Nodes of service \"linker-gateway\": " + strings.Join(svc.Nodes(), ",") + "."
		})
		time.Sleep(time.Second)
	}
}
