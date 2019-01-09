package svc

import (
	"fmt"
	ilog "github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server"
	"net/http"
)

type ServiceRPC struct {
	server.NodeID
}

func (svc ServiceRPC) Keepalive(gateInfo *proto.KeepaliveGatewayInformation, serviceInfo *proto.KeepaliveServiceInformation) error {
	ilog.Infof2("Keepalive from gateway %v.", gateInfo.NodeID.String())
	*serviceInfo = proto.KeepaliveServiceInformation{
		NodeID: svc.NodeID,
	}
	return nil
}

func (svc ServiceRPC) PushMessage(msg *proto.MessagePushArguments, reply *proto.MessagePushResult) error {
	return fmt.Errorf("Message pushing not avaliable.")
}

func ServeRPC() error {
	mux, err := NewServiceServeMux()

	if err != nil {
		err = fmt.Errorf("Error occurs when create mux (%v)", err.Error())
	}

	// Serve RPC
	switch Config.Endpoint.Scheme {
	case "tcp":
		httpServer := &http.Server{
			Addr:    Config.Endpoint.AuthorityString(),
			Handler: mux,
		}
		ilog.Infof0("RPC Serve at %v", Config.Endpoint.String())
		err = httpServer.ListenAndServe()
	default:
		err = fmt.Errorf("Not supported rpc scheme: %v", Config.Endpoint.Scheme)
	}

	return err
}
