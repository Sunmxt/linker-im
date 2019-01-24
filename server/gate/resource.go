package svc

import (
    "github.com/Sunmxt/linker-im/log"
   	"github.com/Sunmxt/linker-im/server/resource"
)

func RegisterResources() error {
	svcEndpointSet := NewServiceEndpointSetFromFlag(Config.ServiceEndpoints, 10, 50)
	log.Infof0("Register resource \"svc-endpoint\".")
	if err := resource.Registry.Register("svc-endpoint", svcEndpointSet); err != nil {
		return err
	}
	svcEndpointSet.GoKeepalive(NodeID, Config.KeepalivePeriod.Value)

    hub := &Hub{
        Meta: ConnectMetadata{
            Timeout: Config.ActiveTimeout.Value,
        }
    }
    log.Info0("Register resource \"hub\"")
	if err := resource.Registry.Register("hub", hub); err != nil {
		return err
	}

	return nil
}
