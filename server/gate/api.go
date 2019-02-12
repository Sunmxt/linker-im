package gate

import (
	"encoding/base64"
	"fmt"
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server"
	sc "github.com/Sunmxt/linker-im/server/svc/client"
	gmux "github.com/gorilla/mux"
	"net/http"
	"strconv"
)

// API
func EntityList(w http.ResponseWriter, req *http.Request) {
	var client *sc.ServiceClient
	var rpcErr error
	vars := gmux.Vars(req)
	ctx, err := NewRequestContext(w, req, nil)
	if err != nil {
		return
	}
	ctx.Version = 1
	entity, ok := vars["entity"]
	if !ok {
		ctx.ResponseError(proto.SERVER_INTERNAL_ERROR, "Variable \"entity\" not found.")
	}
	switch entity {
	case "namespace":
		if client, err = ctx.BeginRPC(); err != nil {
			break
		}
		ctx.Data, rpcErr = client.ListNamespace()
	case "user":
		if client, err = ctx.BeginRPC(); err != nil {
			break
		}
		ctx.Data, rpcErr = client.ListUser(ctx.Namespace)
	case "group":
		if client, err = ctx.BeginRPC(); err != nil {
			break
		}
		ctx.Data, rpcErr = client.ListGroup(ctx.Namespace)
	default:
		ctx.StatusCode = 400
		ctx.ResponseError(proto.INVALID_ARGUMENT, "Invalid entity \""+entity+"\"")
		return
	}
	ctx.EndRPC(rpcErr)
	if rpcErr != nil {
		err = rpcErr
	}
	if err != nil {
		if authErr, isAuthErr := err.(server.AuthError); !isAuthErr {
			log.Error("RPC Error: " + err.Error())
			ctx.ResponseError(proto.SERVER_INTERNAL_ERROR, "(rpc failure) "+err.Error())
		} else {
			ctx.ResponseError(proto.ACCESS_DEINED, authErr.Error())
		}
		return
	}
	ctx.ResponseError(proto.SUCCEED, "")
}

func EntityAlter(w http.ResponseWriter, req *http.Request) {
	var client *sc.ServiceClient
	var rpcErr error
	vars, ireq := gmux.Vars(req), proto.EntityAlterV1{}
	ctx, err := NewRequestContext(w, req, &ireq)
	if err != nil {
		return
	}
	ctx.Version = 1
	entity, ok := vars["entity"]
	if !ok {
		ctx.ResponseError(proto.SERVER_INTERNAL_ERROR, "Variable \"entity\" not found.")
	}
	switch entity {
	case "namespace":
		if client, err = ctx.BeginRPC(); err != nil {
			break
		}
		if req.Method == "POST" {
			rpcErr = client.AddNamespace(ireq.Entities)
		} else {
			rpcErr = client.DeleteNamespace(ireq.Entities)
		}
	case "user":
		if client, err = ctx.BeginRPC(); err != nil {
			break
		}
		if req.Method == "POST" {
			rpcErr = client.AddUser(ctx.Namespace, ireq.Entities)
		} else {
			rpcErr = client.DeleteUser(ctx.Namespace, ireq.Entities)
		}
	case "group":
		if client, err = ctx.BeginRPC(); err != nil {
			break
		}
		if req.Method == "POST" {
			rpcErr = client.AddGroup(ctx.Namespace, ireq.Entities)
		} else {
			rpcErr = client.DeleteGroup(ctx.Namespace, ireq.Entities)
		}
	default:
		ctx.StatusCode = 400
		ctx.ResponseError(proto.INVALID_ARGUMENT, "Invalid entity \""+entity+"\"")
		return
	}
	ctx.EndRPC(rpcErr)
	if rpcErr != nil {
		err = rpcErr
	}
	if err != nil {
		if authErr, isAuthErr := err.(server.AuthError); !isAuthErr {
			log.Error("RPC Error: " + err.Error())
			ctx.ResponseError(proto.SERVER_INTERNAL_ERROR, "(rpc failure) "+err.Error())
		} else {
			ctx.ResponseError(proto.ACCESS_DEINED, authErr.Error())
		}
		return
	}
	ctx.ResponseError(proto.SUCCEED, "")
}

func PushMessage(w http.ResponseWriter, req *http.Request) {
	ireq := proto.MessagePushV1{}
	ctx, err := NewRequestContext(w, req, &ireq)
	if err != nil {
		return
	}
	enc, session, err := ctx.ParseAndGetMessagingClientTuple()
	if err != nil {
		return
	}
	ctx.Version = 1
	if ireq.Msgs == nil || len(ireq.Msgs) < 1 {
		ctx.Data = make([]proto.MessageIdentifier, 0)
	} else {
		if enc == "b64" {
			for idx := range ireq.Msgs {
				bin, err := base64.StdEncoding.DecodeString(ireq.Msgs[idx].Raw)
				if err != nil {
					ctx.ResponseError(proto.INVALID_ARGUMENT, fmt.Sprintf("Invalid base64 string at message %v.", idx))
					return
				}
				ireq.Msgs[idx].Raw = string(bin)
			}
		}
		if ctx.Data, err = gate.push(session, ireq.Msgs); err != nil {
			if authErr, isAuthErr := err.(server.AuthError); !isAuthErr {
				log.Error("RPC Error: " + err.Error())
				ctx.ResponseError(proto.SERVER_INTERNAL_ERROR, "(rpc failure) "+err.Error())
			} else {
				ctx.ResponseError(proto.ACCESS_DEINED, authErr.Error())
			}
			return
		}
	}
	ctx.ResponseError(proto.SUCCEED, "")
}

func PullMessage(w http.ResponseWriter, req *http.Request) {
	var (
		enc, usr string
		timeout  int
		conn     *Connection
		msg      []proto.Message
	)

	ctx, err := NewRequestContext(w, req, nil)
	if err != nil {
		return
	}
	ctx.Version = 1

	var bulk int
	if bulks, ok := ctx.Req.Form["bulk"]; ok && len(bulks) > 0 {
		bulkv, err := strconv.ParseInt(bulks[0], 10, 32)
		if err != nil {
			ctx.ResponseError(proto.INVALID_ARGUMENT, err.Error())
			return
		}
		bulk = int(bulkv)
	} else {
		bulk = -1
	}

	if enc, usr, err = ctx.ParseAndGetMessagingClientTuple(); err != nil {
		return
	}

	if ctx.EnableTimeout {
		timeout = int(ctx.Timeout)
	} else {
		timeout = -1
	}

	if conn, err = gate.Hub.Connect(usr, ConnectMetadata{
		Proto:   PROTO_HTTP,
		Remote:  req.RemoteAddr,
		Timeout: timeout,
	}); err != nil {
		ctx.ResponseError(proto.SERVER_INTERNAL_ERROR, err.Error())
		return
	}
	if bulk < -1 {
		msg = make([]proto.Message, 0, 1)
	} else {
		msg = make([]proto.Message, 0, bulk)
	}

	msg = conn.Receive(msg, bulk, bulk, timeout)
	resp := make([]interface{}, 0, len(msg))
	if enc == "b64" {
		for idx := range msg {
			msg[idx].Body.Raw = base64.StdEncoding.EncodeToString([]byte(msg[idx].Body.Raw))
		}
	}
	for idx := range msg {
		resp = append(resp, msg[idx])
	}
	ctx.Data = resp
	ctx.Finalize()
}

func (g *Gate) InitHTTP() error {
	g.Router = gmux.NewRouter()

	log.Info0("Register HTTP endpoint for entity modification \"/namespace\"")
	g.Router.HandleFunc("/v1/{entity:namespace|group|user}", EntityList).Methods("GET")
	g.Router.HandleFunc("/v1/{entity:namespace|group|user}", EntityAlter).Methods("POST", "DELETE")

	log.Info0("Register HTTP endpoint \"msg\"")
	g.Router.HandleFunc("/v1/msg", PullMessage).Methods("GET")
	g.Router.HandleFunc("/v1/msg", PushMessage).Methods("POST")

	return nil
}

func (g *Gate) ServeHTTP() {
	log.Infof0("Create HTTP Server. Endpoint is \"" + g.config.APIEndpoint.String() + "\"")
	g.HTTP = &http.Server{
		Addr: g.config.APIEndpoint.String(),
		Handler: log.TagLogHandler(g.Router, map[string]interface{}{
			"entity": "http",
		}),
	}

	log.Info0("Serving HTTP API...")
	if err := g.HTTP.ListenAndServe(); err != nil {
		g.fatal <- err
		log.Fatal("HTTP Server failure: " + err.Error())
	}

	log.Info0("HTTP Server exiting...")
}
