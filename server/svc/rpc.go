package svc

import (
	"errors"
	"fmt"
	ilog "github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server"
	"net/http"
    "net/rpc"
)

// Errors

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

func (svc ServiceRPC) Subscribe(args *proto.SubscribeArguments, reply *string) error {
	return nil
}

func (svc ServiceRPC) logError(err error) {
	if err == nil {
		return
	}
	switch err.(type) {
	case server.AuthError:
		svc.log.Info1("Resource authorization error: " + err.Error())
	default:
		svc.log.Info0("RPC Error: " + err.Error())
	}
}

func (svc ServiceRPC) EntityList(args *proto.EntityListArguments, reply *proto.EntityListReply) error {
    var err error
    switch args.Type {
    case proto.ENTITY_NAMESPACE:
        reply.Entities, err = service.Model.ListNamespace()
    case proto.ENTITY_GROUP:
        reply.Entities, err = service.Model.ListGroup(args.Namespace)
    case proto.ENTITY_USER:
        reply.Entities, err = service.Model.ListUser(args.Namespace)
    default:
        reply.Msg = fmt.Sprintf("Unknown entity: %v", args.Type)
    }
    return err
}

func (svc ServiceRPC) EntityAlter(args *proto.EntityAlterArguments, reply *string) error {
    var err error
    if args.Operation != proto.ENTITY_ADD && args.Operation != proto.ENTITY_DEL {
        *reply = fmt.Sprintf("Unknown operation: %v", args.Operation)
        return nil
    }

    switch args.Type {
    case proto.ENTITY_NAMESPACE:
        if len(args.Entities) < 1 {
            break
        }
        if args.Operation == proto.ENTITY_ADD { 
            mapping, empty := make(map[string]*NamespaceMetadata, len(args.Entities)), NewDefaultNamespaceMetadata()
            for idx := range args.Entities {
                mapping[args.Entities[idx]] = empty
            }
            err = service.Model.SetNamespaceMetadata(mapping, true)
        } else {
            err = service.Model.DeleteNamespaceMetadata(args.Entities)
        }

    case proto.ENTITY_GROUP:
        if len(args.Entities) < 2 {
            break
        }
        if args.Operation == proto.ENTITY_ADD {
            mapping, empty := make(map[string]*GroupMetadata, len(args.Entities)-1), NewDefaultGroupMetadata()
            for idx := range args.Entities[1:] {
                mapping[args.Entities[idx]] = empty
            }
            err = service.Model.SetGroupMetadata(args.Entities[0], mapping, true)
        } else {
            err = service.Model.DeleteGroupMetadata(args.Entities[0], args.Entities[1:])
        }

    case proto.ENTITY_USER:
        if len(args.Entities) < 2 {
            break
        }
        if args.Operation == proto.ENTITY_ADD {
            mapping, empty := make(map[string]*UserMetadata, len(args.Entities)-1), NewDefaultUserMetadata()
            for idx := range args.Entities[1:] {
                mapping[args.Entities[idx]] = empty
            }
            err = service.Model.SetUserMetadata(args.Entities[0], mapping, true)
        } else {
            err = service.Model.DeleteUserMetadata(args.Entities[0], args.Entities[1:])
        }

    default:
        *reply = fmt.Sprintf("Unknown entity: %v", args.Type)
    }
    return err
}

func (svc *Service) InitRPC() error {
    rpcServer := rpc.NewServer()
    rpcRuntime := ServiceRPC{
        NodeID: svc.ID,
        log: ilog.NewLogger(),
    }
    rpcRuntime.log.Fields["entity"] = "rpc"

    rpcServer.Register(rpcRuntime)

    // Mux
    healthMux := http.NewServeMux()
    healthMux.HandleFunc("/", Healthz)

    svc.RPCRouter = http.NewServeMux()
    // Health check
    ilog.Info0("Register RPC health-check endpoint \"/healthz\"")
	svc.RPCRouter.Handle("/healthz", ilog.TagLogHandler(healthMux, map[string]interface{}{
		"entity": "health-check",
	}))

	// RPC
    ilog.Info0("Register RPC endpoint \"" + proto.RPC_PATH + "\"")
	svc.RPCRouter.Handle(proto.RPC_PATH, rpcServer)

    if svc.Config.Endpoint.Scheme != "tcp" {
		return errors.New("Not supported rpc scheme: " + svc.Config.Endpoint.Scheme)
    }

	ilog.Info0("Initialize RPC Server at \"" +  svc.Config.Endpoint.String() + "\".")
    svc.RPC = &http.Server{
		Addr:    svc.Config.Endpoint.AuthorityString(),
		Handler: svc.RPCRouter,
    }

    return nil
}

func (svc *Service) ServeRPC() {
    ilog.Info0("RPC Serving...")
	if err := svc.RPC.ListenAndServe(); err != nil {
        ilog.Error("RPC Server failure: " + err.Error())
        svc.fatal <- err
    }
    ilog.Info0("RPC Stopping...")
}
