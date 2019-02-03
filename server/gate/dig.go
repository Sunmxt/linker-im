package gate

import (
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/server/dig"
    "github.com/Sunmxt/linker-im/proto"
	"strings"
	"time"
)

func (g *Gate) Dig() {
	var (
		svc dig.Service
		err error
	)

	log.Info0("Start node discovery.")
    g.discover = make(chan *dig.Notification)

	for {
		svc, err = g.Dig.Service(proto.DIG_GATE_SERVICE_NAME)
		if err != nil {
			log.Error("Cannot open service \""+proto.DIG_GATE_SERVICE_NAME+"\": " + err.Error())
		} else if svc == nil {
			log.Error("Nil value of service \"" + proto.DIG_GATE_SERVICE_NAME + "\".")
		} else {
			break
		}
		time.Sleep(time.Second)
	}
	log.Info0("Service \"" + proto.DIG_GATE_SERVICE_NAME + "\" opened. Start node discovery.")

	g.Node = &dig.Node{
		Name: "gateway-" + g.ID.String(),
		Metadata: map[string]string{
			"linker-rpc":    g.config.RPCPublishEndpoint.String(),
			"linker-nodeid": NodeID.String(),
		},
		Timeout: 300,
	}
	svc.Publish(g.Node)
	log.Info0("Publish gateway node \"" + g.Node.Name + "\" of service \"" + proto.DIG_GATE_SERVICE_NAME + "\".")

	for {
		changed, err := g.Dig.Poll(g.discover)
		if err != nil {
			log.Error("Dig polling failure: " + err.Error())
		}
		if changed {
			log.Info2("Dig state changed.")
			log.Info0("Nodes of service \"" + proto.DIG_GATE_SERVICE_NAME + "linker-gateway\": " + strings.Join(svc.Nodes(), ",") + ".")
		}
		log.DebugLazy(func() string {
			return "Nodes of service \"" + proto.DIG_GATE_SERVICE_NAME + "linker-gateway\": " + strings.Join(svc.Nodes(), ",") + "."
		})
		time.Sleep(time.Second)
	}

	log.Info0("Node discovery stopping...")

}
