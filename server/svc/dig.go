package svc

import (
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server"
	"github.com/Sunmxt/linker-im/server/dig"
	"runtime"
	"strings"
	"time"
)

func (s *Service) openDigService(name string) dig.Service {
	var (
		svc dig.Service
		err error
	)

	for {
		svc, err = s.Reg.Service(name)
		if err != nil {
			log.Error("[Dig] Cannot open service \"" + name + "\": " + err.Error())
		} else if svc == nil {
			log.Error("[Dig] Nil value of service \"" + name + "\".")
		} else {
			break
		}
		time.Sleep(time.Second)
	}
	return svc
}

func (svc *Service) gateOp(notify *dig.Notification, oper func(rawID, rpc string, id server.NodeID)) {
	var (
		ID  server.NodeID
		rpc string
	)
	rawID, ok := notify.Node.Metadata["linker-nodeid"]
	if !ok {
		log.Info2("[Dig] ID of node \"" + notify.Node.Name + "\" is missing. Skip.")
		return
	}
	rpc, ok = notify.Node.Metadata["linker-rpc"]
	if !ok {
		log.Info2("[Dig] RPC endpoint of node \"" + notify.Node.Name + "\" is missing. Skip.")
		return
	}
	if err := ID.FromString(rawID); err != nil {
		log.Warn("[Dig] Invalid ID of node \"" + notify.Node.Name + "\". skip.")
		return
	}
	oper(rawID, rpc, ID)
}

func (svc *Service) addGate(notify *dig.Notification) {
	svc.gateOp(notify, func(rawID, rpc string, id server.NodeID) {
		_, loaded := svc.gateNode.Load(rawID)
		if !loaded {
			_, loaded = svc.gateNode.LoadOrStore(rawID, OpenGateNode(id, notify.Node.Name, rpc, proto.RPC_PATH, runtime.NumCPU(), runtime.NumCPU()))
			if !loaded {
				log.Info0("[Dig] Add node \"" + notify.Node.Name + "\" with ID \"" + rawID + "\" to load balancer. Endpoint is \"" + rpc + "\".")
			}
		}
	})
}

func (svc *Service) removeGate(notify *dig.Notification) {
	svc.gateOp(notify, func(rawID, rpc string, id server.NodeID) {
		log.Info0("[Dig] Remove node \"" + notify.Node.Name + "\" with ID \"" + rawID + "\" to load balancer. Endpoint is \"" + rpc + "\".")
		svc.gateNode.Delete(rawID)
	})
}

func (svc *Service) Discover() {
	var (
		gateSvc, svcSvc dig.Service
	)

	log.Info0("[Dig] start.")

	gateSvc = svc.openDigService(proto.DIG_GATE_SERVICE_NAME)
	log.Info0("[Dig] Service \"" + proto.DIG_GATE_SERVICE_NAME + "\" opened.")

	svcSvc = svc.openDigService(proto.DIG_SERVICE_NAME)
	log.Info0("[Dig] Service \"" + proto.DIG_SERVICE_NAME + "\" opened.")

	svc.Node = &dig.Node{
		Name: "svc-" + svc.ID.String(),
		Metadata: map[string]string{
			"linker-rpc":    svc.Config.RPCPublish.String(),
			"linker-nodeid": svc.ID.String(),
			"linker-role":   "svc",
		},
		Timeout: 3,
	}
	svcSvc.Publish(svc.Node)
	log.Info0("[Dig] Publish node \"" + svc.Node.Name + "\" of service \"" + proto.DIG_SERVICE_NAME + "\".")
	for {
		changed, err := svc.Reg.Poll(func(notify *dig.Notification) {
			switch notify.Event {
			case dig.EVENT_NODE_FOCUS:
				log.Info2("[Dig] Focus on node \"" + notify.Name + "\"")

			case dig.EVENT_SVC_NODE_FOUND:
				log.Info0("[Dig] Node \"" + notify.Name + "\" of service \"" + notify.Service.Name() + "\" discovered.")

			case dig.EVENT_NODE_LOST:
				log.Info0("[Dig] Node \"" + notify.Name + "\" lost.")

			case dig.EVENT_NODE_METADATA_KEY_ADD:
				role, ok := notify.Node.Metadata["linker-role"]
				if ok && role == "gate" && (notify.Name == "linker-nodeid" || notify.Name == "linker-rpc" || notify.Name == "linker-role") {
					svc.addGate(notify)
				}

			case dig.EVENT_NODE_METADATA_KEY_CHANGED:
				role, ok := notify.Node.Metadata["linker-role"]
				if ok && role == "gate" && (notify.Name == "linker-nodeid" || notify.Name == "linker-rpc" || notify.Name == "linker-role") {
					svc.removeGate(notify)
					svc.addGate(notify)
				}

			case dig.EVENT_NODE_METADATA_KEY_DEL:
				role, ok := notify.Node.Metadata["linker-role"]
				if ok && role == "gate" && (notify.Name == "linker-nodeid" || notify.Name == "linker-rpc" || notify.Name == "linker-role") {
					svc.removeGate(notify)
				}
			}
		})
		if err != nil {
			log.Error("[Dig] polling failure: " + err.Error())
		}
		if changed {
			log.Info2("[Dig] state changed.")
			log.Info0("[Dig] Node of service \"" + proto.DIG_SERVICE_NAME + "\": " + strings.Join(svcSvc.Nodes(), " ,") + ".")
			log.Info0("[Dig] Node of service \"" + proto.DIG_GATE_SERVICE_NAME + "\": " + strings.Join(gateSvc.Nodes(), " ,") + ".")
		}
		time.Sleep(time.Second)
	}
	log.Info0("[Dig] stopping...")
}
