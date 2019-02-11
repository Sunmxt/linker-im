package svc

import (
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server/dig"
	"strings"
	"time"
)

func (svc *Service) Discover() {
	var (
		isvc dig.Service
		err  error
	)

	log.Info0("Start node discovery")

	for {
		isvc, err = svc.Reg.Service(proto.DIG_SERVICE_NAME)
		if err != nil {
			log.Error("Cannot open service \"" + proto.DIG_SERVICE_NAME + "\": " + err.Error())
		} else if svc == nil {
			log.Error("Nil value of service \"" + proto.DIG_SERVICE_NAME + "\".")
		} else {
			break
		}
		time.Sleep(time.Second)
	}
	log.Info0("Service \"" + proto.DIG_SERVICE_NAME + "\" opened. Start node discovery.")
	svc.Node = &dig.Node{
		Name: "svc-" + svc.ID.String(),
		Metadata: map[string]string{
			"linker-rpc":    svc.Config.RPCPublish.String(),
			"linker-nodeid": svc.ID.String(),
			"linker-role":   "svc",
		},
	}
	isvc.Publish(svc.Node)
	log.Info0("Publish node \"" + svc.Node.Name + "\" of service \"" + proto.DIG_SERVICE_NAME + "\".")
	for {
		changed, err := svc.Reg.Poll(nil)
		if err != nil {
			log.Error("Dig polling failure: " + err.Error())
		}
		if changed {
			log.Info2("Dig state changed.")
			log.Info0("Node of service \"" + proto.DIG_SERVICE_NAME + "\": " + strings.Join(isvc.Nodes(), " ,") + ".")
		}
		log.DebugLazy(func() string {
			return "Nodes of service \"" + proto.DIG_SERVICE_NAME + "\": " + strings.Join(isvc.Nodes(), ",") + "."
		})
		time.Sleep(time.Second)
	}
}
