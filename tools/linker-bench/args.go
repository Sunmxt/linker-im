package main

import (
	"flag"
	"github.com/Sunmxt/linker-im/utils/cmdline"
	"log"
)

type BenchmarkConfigure struct {
	PushMode     bool
	Concurrency  uint
	Request      uint
	UserPrefix   string
	GroupPrefix  string
	GroupCount   uint
	Namespace    string
	GateEndpoint *cmdline.NetEndpointValue
}

func parseConfigure() *BenchmarkConfigure {
	config := &BenchmarkConfigure{}

	gateEndpoint, err := cmdline.NewNetEndpointValueDefault([]string{"tcp", "http"}, "127.0.0.1:12360")
	if err != nil {
		log.Panicln(err.Error())
		return nil
	}
	config.GateEndpoint = gateEndpoint

	flag.BoolVar(&config.PushMode, "push", false, "Push mode.")
	flag.UintVar(&config.Concurrency, "concurrency", 1, "Concurrency level. Number of requests to push in push mode. Number of clients in pull mode.")
	flag.UintVar(&config.Request, "request", 1, "Number of messages to send or receive.")
	flag.StringVar(&config.UserPrefix, "user-prefix", "test", "Name prefix of generated user.")
	flag.StringVar(&config.GroupPrefix, "group-prefix", "test", "Group prefix of generated user.")
	flag.UintVar(&config.GroupCount, "group-count", 1, "Number of groups to send or receive.")
	flag.StringVar(&config.Namespace, "namespace", "test", "Namespace.")
	flag.Var(config.GateEndpoint, "gate", "linker-gate endpoint.")

	flag.Parse()

	if fl := flag.Lookup("group-count"); fl != nil {
		v := fl.Value.String()
		if v == "" || v == "0" {
			log.Println("Group count is too small. set to 1.")
			fl.Value.Set("1")
		}
	}

	if fl := flag.Lookup("concurrency"); fl != nil {
		v := fl.Value.String()
		if v == "" || v == "0" {
			log.Println("Concurrency is too small. set to 1.")
			fl.Value.Set("1")
		}
	}

	log.Println("Configure:")
	flag.VisitAll(func(fl *flag.Flag) {
		log.Println("\t-" + fl.Name + "=" + fl.Value.String())
	})

	return config
}
