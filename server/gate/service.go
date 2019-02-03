package gate

import (
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/server/dig"
	"github.com/gomodule/redigo/redis"
)

func (g *Gate) InitService() error {
	var err error

	log.Info0("Create service load balancer.")
    g.LB = NewLB()

	log.Info0("Redis connection pooling.")
	g.Redis = &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", g.config.RedisEndpoint.AuthorityString())
		},
		MaxIdle:         int(g.config.RedisPoolIdleMax.Value),
		MaxActive:       int(g.config.RedisPoolActiveMax.Value),
		Wait:            true,
		MaxConnLifetime: 0,
		IdleTimeout:     0,
	}

	log.Info0("Initialize node discovery.")
	if g.Dig, err = dig.Connect("redis", g.Redis, g.config.RedisPrefix.Value); err != nil {
		return err
	}

	log.Info0("Initialize hub.")
	g.Hub = NewHub(ConnectMetadata{
		Timeout: int(g.config.ActiveTimeout.Value),
	})

	return nil
}
