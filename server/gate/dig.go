package gate

import (
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server"
	"github.com/Sunmxt/linker-im/server/dig"
	"runtime"
	"strings"
	"time"
)

func (g *Gate) openDigService(name string) dig.Service {
	var (
		svc dig.Service
		err error
	)

	for {
		svc, err = g.Dig.Service(proto.DIG_GATE_SERVICE_NAME)
		if err != nil {
			log.Error("Cannot open service \"" + proto.DIG_GATE_SERVICE_NAME + "\": " + err.Error())
		} else if svc == nil {
			log.Error("Nil value of service \"" + proto.DIG_GATE_SERVICE_NAME + "\".")
		} else {
			break
		}
		time.Sleep(time.Second)
	}
	return svc
}

func (g *Gate) addServiceNode(notify *dig.Notification) {
	var (
		ID  server.NodeID
		rpc string
	)
	rawID, ok := notify.Node.Metadata["linker-nodeid"]
	if !ok {
		log.Info0("ID of node \"" + notify.Node.Name + "\" is missing. Skip.")
	}
	rpc, ok = notify.Node.Metadata["linker-rpc"]
	if !ok {
		log.Info0("RPC endpoint of node \"" + notify.Node.Name + "\" is missing. Skip.")
	}
	if err := ID.FromString(rawID); err != nil {
		log.Warn("Invalid ID of node \"" + notify.Node.Name + "\". skip.")
	}
	log.Info0("Add node \"" + notify.Node.Name + "\" with ID \"" + rawID + "\" to load balancer. Endpoint is \"" + rpc + "\".")
	g.LB.AddNode(notify.Node.Name, OpenServiceNode(ID, notify.Node.Name, rpc, proto.RPC_PATH, runtime.NumCPU(), runtime.NumCPU()))
}

func (g *Gate) removeServiceNode(notify *dig.Notification) {
	log.Info0("Remove node \"" + notify.Node.Name)
	g.LB.RemoveNode(notify.Node.Name)
}

func (g *Gate) Discover() {
	var (
		svc dig.Service
		err error
	)

	log.Info0("Start node discovery.")

	svc = g.openDigService(proto.DIG_GATE_SERVICE_NAME)
	log.Info0("Service \"" + proto.DIG_GATE_SERVICE_NAME + "\" opened. Start node discovery.")
	g.openDigService(proto.DIG_SERVICE_NAME)
	log.Info0("Service \"" + proto.DIG_SERVICE_NAME + "\" opened. Start node discovery.")

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
		changed, err := g.Dig.Poll(func(notify *dig.Notification) {
			switch notify.Event {
			case dig.EVENT_NODE_FOCUS:
				log.Info0("Watch node \"" + notify.Node.Name + "\" of service \"" + notify.Name + "\".")
			case dig.EVENT_SVC_NODE_FOUND:
				log.Info0("Discover node \"" + notify.Node.Name + "\" of service \"" + notify.Name + "\".")
			case dig.EVENT_NODE_METADATA_KEY_ADD:
				if notify.Name == "linker-nodeid" || notify.Name == "linker-rpc" {
					g.addServiceNode(notify)
				}
			case dig.EVENT_NODE_METADATA_KEY_CHANGED:
				if notify.Name == "linker-nodeid" || notify.Name == "linker-rpc" {
					g.removeServiceNode(notify)
					g.addServiceNode(notify)
				}
			case dig.EVENT_NODE_METADATA_KEY_DEL:
				if notify.Name == "linker-nodeid" || notify.Name == "linker-rpc" {
					g.removeServiceNode(notify)
				}
			}

		})
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
