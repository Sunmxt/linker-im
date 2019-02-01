package svc

import (
    "github.com/Sunmxt/linker-im/log"
    "github.com/gomodule/redigo/redis"
    "runtime"
)

func (svc *Service) InitService() error {
    log.Info0("Initizlize redis connection pool.")
	svc.Redis = &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", svc.Config.RedisEndpoint.AuthorityString())
		},
		MaxIdle:         runtime.NumCPU() * 2,
		MaxActive:       runtime.NumCPU() * 2,
		Wait:            true,
		MaxConnLifetime: 0,
		IdleTimeout:     0,
	}

    log.Info0("Initialize model.")
    svc.Model = NewModel(svc.Redis, svc.Config.RedisPrefix.Value)

    return nil
}
