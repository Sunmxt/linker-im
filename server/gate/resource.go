package gate

import (
	"errors"
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/server/dig"
	"github.com/Sunmxt/linker-im/server/resource"
)

func GetHub() (*Hub, error) {
	raw, err := resource.Registry.Access("hub")
	if err != nil {
		return nil, err
	}
	hub, ok := raw.(*Hub)
	if !ok {
		return nil, errors.New("Invalid Hub type.")
	}
	return hub, nil
}

func RegisterResources() error {
	svcEndpointSet := NewServiceEndpointSetFromFlag(Config.ServiceEndpoints, 10, 50)
	log.Info0("Register resource \"svc-endpoint\".")
	if err := resource.Registry.Register("svc-endpoint", svcEndpointSet); err != nil {
		return err
	}
	svcEndpointSet.GoKeepalive(NodeID, Config.KeepalivePeriod.Value)

	reg, err := dig.Connect("redis", Config.RedisEndpoint.AuthorityString(), Config.RedisPrefix.Value, 100, 100)
	if err != nil {
		return err
	}
	log.Info0("Register resource \"dig\".")
	if err := resource.Registry.Register("dig", reg); err != nil {
		return err
	}
	go discover(reg)

	hub := &Hub{
		Meta: ConnectMetadata{
			Timeout: int(Config.ActiveTimeout.Value),
		},
	}
	log.Info0("Register resource \"hub\"")
	if err := resource.Registry.Register("hub", hub); err != nil {
		return err
	}

	return nil
}
