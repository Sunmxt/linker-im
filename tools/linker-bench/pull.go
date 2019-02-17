package main

import (
	"bytes"
	"encoding/json"
	"github.com/Sunmxt/linker-im/proto"
	"io"
	"log"
	"net/http"
	"strconv"
)

func DumpResponse(resp *http.Response) {
	log.Fatalf("Header: %v", resp.Header)
	buf := make([]byte, resp.ContentLength)
	if _, err := io.ReadFull(resp.Body, buf); err != nil {
		log.Fatalf("HTTP body dump failure: " + err.Error())
		return
	}
	log.Fatalln("Body: " + string(buf))
}

func GoPull(config *BenchmarkConfigure) {
	log.Println("Work in pull mode.")
	log.Println("Start subscribe.")

	urlSub, sub := "http://"+config.GateEndpoint.AuthorityString()+"/v1/sub?ns="+config.Namespace, proto.Subscription{}
	for i := uint(0); i < config.GroupCount; i++ {
		sub.Group = config.GroupPrefix + strconv.FormatUint(uint64(i), 10)
		for j := uint(0); j < config.Concurrency; i++ {
			sub.Session = config.UserPrefix + strconv.FormatUint(uint64(j), 10)
			buf, err := json.Marshal(sub)
			if err != nil {
				log.Fatalln("Json marshal failure: " + err.Error())
				return
			}
			resp, err := http.Post(urlSub, "application/json", bytes.NewReader(buf))
			if err != nil {
				log.Fatalln(err.Error())
				return
			}
			if resp.StatusCode != 200 {
				log.Fatalf("Subscription HTTP code %v\b.", resp.StatusCode)
				DumpResponse(resp)
				return
			}
		}
	}
}
