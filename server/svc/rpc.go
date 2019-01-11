package svc

import (
	"errors"
	"fmt"
	ilog "github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server"
	"github.com/Sunmxt/linker-im/server/resource"
	"net/http"
)

type ServiceRPC struct {
	server.NodeID
	log *ilog.Logger
}

func (svc ServiceRPC) Keepalive(gateInfo *proto.KeepaliveGatewayInformation, serviceInfo *proto.KeepaliveServiceInformation) error {
	ilog.Infof2("Keepalive from gateway %v.", gateInfo.NodeID.String())
	*serviceInfo = proto.KeepaliveServiceInformation{
		NodeID: svc.NodeID,
	}
	return nil
}

// Push message sequences.
func (svc ServiceRPC) Push(msg *proto.MessagePushArguments, reply *proto.MessagePushResult) error {
	return fmt.Errorf("Message pushing not avaliable.")
}

func (svc ServiceRPC) logError(err error) {
	if err == nil {
		return
	}
	switch err.(type) {
	case resource.ResourceAuthError:
		svc.log.Info1("Resource authorization error: " + err.Error())
	default:
		svc.log.Info0("RPC Error: " + err.Error())
	}
}

// Append session namespace.
func (svc ServiceRPC) NamespaceAdd(args *proto.NamespaceArguments, reply *proto.Dummy) error {
	var res interface{}
	var err error

	defer func() {
		svc.logError(err)
	}()

	res, err = resource.Registry.AuthAccess("namespace", nil)
	if err != nil {
		if _, ok := err.(resource.ResourceAuthError); ok {
			svc.log.Infof1("")
			return nil
		} else {
			return err
		}
	}

	sessionNamespace, ok := res.(*SessionNamespace)
	if !ok {
		return errors.New("Resource has invalid type.")
	}

	if err = sessionNamespace.Append(args.Names); err != nil {
		return err
	}

	return nil
}

func (svc ServiceRPC) NamespaceList(args *proto.Dummy, reply *proto.NamespaceListReply) error {
	return fmt.Errorf("Not avaliable.")
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
