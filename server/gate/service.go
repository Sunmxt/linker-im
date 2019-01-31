package gate

import (
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/server/dig"
)

func (g *Gate) InitService() error {
	var err error

	g.Service = NewServiceEndpointSetFromFlag(g.config.ServiceEndpoints, 10, 50)
	log.Info0("Create service endpoint set.")
	g.Service.GoKeepalive(g.ID, g.config.KeepalivePeriod.Value)

	log.Info0("Initialize node discovery.")
	if g.Dig, err = dig.Connect("redis", g.config.RedisEndpoint.AuthorityString(), g.config.RedisPrefix.Value, 100, 100); err != nil {
		return err
	}

	log.Info0("Initialize hub.")
	g.Hub = &Hub{
		Meta: ConnectMetadata{
			Timeout: int(g.config.ActiveTimeout.Value),
		},
	}

	return nil
}
