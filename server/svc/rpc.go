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

func (svc ServiceRPC) Echo(args *string, reply *string) error {
	*reply = *args
	return nil
}

func rpcAuth(op interface{}, namespace, session string) (string, error) {
	var ident string
	sessionMap, err := service.Session.Get(namespace, session)
	if err != nil {
		return "", errors.New("Session failure: " + err.Error())
	}
	if err = service.Auther.Auth(namespace, op, sessionMap); err != nil {
		return "", errors.New("Operation failure: " + err.Error())
	}
	if ident, err = service.Auther.Identifier(namespace, sessionMap); err != nil {
		return "", errors.New("Identifier resolution failure: " + err.Error())
	}
	return ident, nil
}

// Push message sequences.
func (svc ServiceRPC) Push(args *proto.RawMessagePushArguments, reply *proto.MessagePushResult) error {
	ident, err := rpcAuth(args, args.Namespace, args.Session)
	result := make([]proto.PushResult, len(args.Msgs))
	if err != nil {
		reply.IsAuthError = true
		reply.Msg = err.Error()
		return nil
	}
	if err = service.serial.SerializeMessage(ident, args.Msgs, result); err != nil {
		return err
	}
	msgs := make([]proto.Message, len(args.Msgs))
	for idx := range msgs {
		msgs[idx].MessageBody = args.Msgs[idx]
		msgs[idx].MessageIdentifier = result[idx].MessageIdentifier
	}
	service.pushBulk(args.Namespace, msgs, result)
	reply.Replies = result
	return err
}

func (svc ServiceRPC) Subscribe(args *proto.Subscription, reply *string) error {
	ident, err := rpcAuth(args, args.Namespace, args.Session)
	if err != nil {
		*reply = err.Error()
		return nil
	}
	switch args.Op {
	case proto.OP_SUB_ADD:
		return service.Model.Subscribe(args.Namespace, args.Group, []string{ident})
	case proto.OP_SUB_CANCEL:
		return service.Model.Unsubscribe(args.Namespace, args.Group, []string{ident})
	default:
		return errors.New("Invalid operation for subscription.")
	}
}

func (svc ServiceRPC) Connect(conn *proto.ConnectV1, reply *proto.ConnectResultV1) error {
	session, ident := make(map[string]string), ""
	err := service.Auther.Connect(conn.Namespace, conn.Credential, session)
	if err != nil {
		if server.IsAuthError(err) {
			reply.AuthError = err.Error()
			return nil
		}
		return err
	}
	if reply.Session, err = service.Session.Register(conn.Namespace, session); err != nil {
		return err
	}
	if ident, err = service.Auther.Identifier(conn.Namespace, session); err != nil {
		reply.AuthError = "Identifier resolution failure: " + err.Error()
		return nil
	}
	reply.Key = conn.Namespace + "." + ident
	return nil
}

func (svc ServiceRPC) EntityList(args *proto.EntityListArguments, reply *proto.EntityListReply) error {
	var err error
	if _, err = rpcAuth(args, args.Namespace, args.Session); err != nil {
		return err
	}
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
	if err != nil {
		return err
	}
	return nil
}

func (svc ServiceRPC) EntityAlter(args *proto.EntityAlterV1, reply *string) error {
	var err error
	if args.Operation != proto.ENTITY_ADD && args.Operation != proto.ENTITY_DEL {
		*reply = fmt.Sprintf("Unknown operation: %v", args.Operation)
		return nil
	}

	if _, err = rpcAuth(args, args.Namespace, args.Session); err != nil {
		return err
	}
	if args.Entities == nil || len(args.Entities) < 1 {
		return nil
	}
	switch args.Type {
	case proto.ENTITY_NAMESPACE:
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
		if args.Namespace == "" {
			*reply = "Empty namespace."
			return nil
		}
		if args.Operation == proto.ENTITY_ADD {
			mapping, empty := make(map[string]*GroupMetadata, len(args.Entities)-1), NewDefaultGroupMetadata()
			for idx := range args.Entities {
				mapping[args.Entities[idx]] = empty
			}
			err = service.Model.SetGroupMetadata(args.Namespace, mapping, true)
		} else {
			err = service.Model.DeleteGroupMetadata(args.Namespace, args.Entities)
		}

	case proto.ENTITY_USER:
		if args.Namespace == "" {
			*reply = "Empty namespace."
			return nil
		}
		if args.Operation == proto.ENTITY_ADD {
			mapping, empty := make(map[string]*UserMetadata, len(args.Entities)-1), NewDefaultUserMetadata()
			for idx := range args.Entities {
				mapping[args.Entities[idx]] = empty
			}
			err = service.Model.SetUserMetadata(args.Namespace, mapping, true)
		} else {
			err = service.Model.DeleteUserMetadata(args.Namespace, args.Entities)
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
		log:    ilog.NewLogger(),
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

	ilog.Info0("Initialize RPC Server at \"" + svc.Config.Endpoint.String() + "\".")
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
