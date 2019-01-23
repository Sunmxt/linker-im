package svc

import (
	ilog "github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/server/resource"
	"github.com/gomodule/redigo/redis"
	"runtime"
)

var log *ilog.Logger

func init() {
	log = ilog.NewLogger()
	log.Fields["entity"] = "resource"
}

func RegisterResources() error {
	// Redis
	log.Info0("Register resource \"redis\"")
	redisPool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", Config.RedisEndpoint.AuthorityString())
		},
		MaxIdle:         runtime.NumCPU() * 2,
		MaxActive:       runtime.NumCPU() * 2,
		Wait:            true,
		MaxConnLifetime: 0,
		IdleTimeout:     0,
	}
	if err := resource.Registry.Register("redis", redisPool); err != nil {
		log.Infof0("Resource \"redis\" register failure. (%v)", err.Error())
	}

	// model
	log.Info0("Register resource \"model\"")
	model := NewModel(redisPool, Config.RedisPrefix.Value)
	if err := resource.Registry.Register("model", model); err != nil {
		log.Infof0("Resource \"model\" register failure. (%v)", err.Error())
		return err
	}

	return nil
}
