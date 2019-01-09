package svc

import (
	ilog "github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/server/resource"
	"runtime"
)

var log *ilog.Logger

func init() {
	log = ilog.NewLogger()
	log.Fields["entity"] = "resource"
}

func RegisterResources() error {
	log.Infof0("Register resource \"namespace\"")
	sessionNamespace := NewSessionNamespace("tcp", Config.RedisEndpoint.AuthorityString(), Config.RedisPrefix.Value, int(Config.CacheTimeout.Value), runtime.NumCPU()*2, nil)
	if err := resource.Registry.Register("namespace", sessionNamespace); err != nil {
		log.Infof0("Resource \"namespace\" register failure. (%v)", err.Error())
		return err
	}
	return nil
}
