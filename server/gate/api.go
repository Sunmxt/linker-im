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
	vars := gmux.Vars(req)
	ctx, err := NewRequestContext(w, req, nil)
	if err != nil {
		return
	}
	_, session, err := ctx.ParseAndGetMessagingClientTuple()
	if err != nil {
		return
	}
	ctx.Version = 1
	entity, ok := vars["entity"]
	if !ok {
		ctx.ResponseError(proto.SERVER_INTERNAL_ERROR, "Variable \"entity\" not found.")
	}
	defer func() {
		if err != nil {
			if authErr, isAuthErr := err.(server.AuthError); !isAuthErr {
				log.Error("RPC Error: " + err.Error())
				ctx.ResponseError(proto.SERVER_INTERNAL_ERROR, "(rpc failure) "+err.Error())
			} else {
				ctx.ResponseError(proto.ACCESS_DEINED, authErr.Error())
			}
		}
	}()
	if client, err = ctx.BeginRPC(); err != nil {
		return
	}
	switch entity {
	case "namespace":
		ctx.Data, err = client.ListNamespace(session)
	case "user":
		ctx.Data, err = client.ListUser(ctx.Namespace, session)
	case "group":
		ctx.Data, err = client.ListGroup(ctx.Namespace, session)
	}
	ctx.EndRPC(err)
	if err != nil {
		return
	}
	ctx.ResponseError(proto.SUCCEED, "")
}

func EntityAlter(w http.ResponseWriter, req *http.Request) {
	var client *sc.ServiceClient
	vars, ireq := gmux.Vars(req), proto.EntityAlterV1{}
	ctx, err := NewRequestContext(w, req, &ireq)
	if err != nil {
		return
	}
	_, ireq.Session, err = ctx.ParseAndGetMessagingClientTuple()
	if err != nil {
		return
	}
	ctx.Version = 1
	ireq.Namespace = ctx.Namespace
	entity, ok := vars["entity"]
	if !ok {
		ctx.ResponseError(proto.SERVER_INTERNAL_ERROR, "Variable \"entity\" not found.")
		return
	}
	defer func() {
		if err != nil {
			if authErr, isAuthErr := err.(server.AuthError); !isAuthErr {
				log.Error("RPC Error: " + err.Error())
				ctx.ResponseError(proto.SERVER_INTERNAL_ERROR, "(rpc failure) "+err.Error())
			} else {
				ctx.ResponseError(proto.ACCESS_DEINED, authErr.Error())
			}
		}
	}()
	if client, err = ctx.BeginRPC(); err != nil {
		return
	}
	switch entity {
	case "namespace":
		ireq.Type = proto.ENTITY_NAMESPACE
	case "user":
		ireq.Type = proto.ENTITY_USER
	case "group":
		ireq.Type = proto.ENTITY_GROUP
	}
	if req.Method == "POST" {
		ireq.Operation = proto.ENTITY_ADD
	} else {
		ireq.Operation = proto.ENTITY_DEL
	}
	err = client.AlterEntity(&ireq)
	ctx.EndRPC(err)
	if err != nil {
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
		if ctx.Data, err = gate.push(ctx.Namespace, session, ireq.Msgs); err != nil {
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
		enc, s  string
		timeout int
		conn    *Connection
		msg     []proto.Message
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

	if enc, s, err = ctx.ParseAndGetMessagingClientTuple(); err != nil {
		return
	}

	if ctx.EnableTimeout {
		timeout = int(ctx.Timeout)
	} else {
		timeout = -1
	}

	if conn, err = gate.hubConnect(ctx.Namespace, s, ConnectMetadata{
		Proto:   PROTO_HTTP,
		Remote:  req.RemoteAddr,
		Timeout: timeout,
	}); err != nil {
		if !server.IsAuthError(err) {
			log.Error("Hub connect failure: " + err.Error())
			ctx.ResponseError(proto.SERVER_INTERNAL_ERROR, err.Error())
		} else {
			ctx.ResponseError(proto.ACCESS_DEINED, err.Error())
		}
		return
	}
	if bulk > 0 {
		msg = make([]proto.Message, 0, bulk)
	} else {
		msg = make([]proto.Message, 0, 1)
	}

	msg = conn.Receive(msg, bulk, bulk, timeout)
	resp := make([]interface{}, 0, len(msg))
	if enc == "b64" {
		for idx := range msg {
			msg[idx].MessageBody.Raw = base64.StdEncoding.EncodeToString([]byte(msg[idx].MessageBody.Raw))
		}
	}
	for idx := range msg {
		resp = append(resp, msg[idx])
	}
	ctx.Data = resp
	ctx.ResponseError(proto.SUCCEED, "")
}

func Subscribe(w http.ResponseWriter, req *http.Request) {
	var sub proto.Subscription
	ctx, err := NewRequestContext(w, req, &sub)
	if err != nil {
		return
	}
	switch req.Method {
	case "POST":
		sub.Op = proto.OP_SUB_ADD
	case "DELETE":
		sub.Op = proto.OP_SUB_CANCEL
	}
	sub.Namespace = ctx.Namespace
	if err = gate.subscribe(sub); err != nil {
		if !server.IsAuthError(err) {
			log.Error("RPC Error: " + err.Error())
			ctx.ResponseError(proto.SERVER_INTERNAL_ERROR, "(rpc failure) "+err.Error())
		} else {
			ctx.ResponseError(proto.ACCESS_DEINED, err.Error())
		}
	}
	ctx.ResponseError(proto.SUCCEED, "")
}

func Connect(w http.ResponseWriter, req *http.Request) {
	var conn proto.ConnectV1
	var result *proto.ConnectResultV1
	ctx, err := NewRequestContext(w, req, &conn)
	if err != nil {
		return
	}
	conn.Namespace = ctx.Namespace
	if result, err = gate.connect(&conn); err != nil {
		if !server.IsAuthError(err) {
			log.Error("RPC Error: " + err.Error())
			ctx.ResponseError(proto.SERVER_INTERNAL_ERROR, "(rpc failure) "+err.Error())
		} else {
			ctx.ResponseError(proto.ACCESS_DEINED, err.Error())
		}
	}
	ctx.Data = result
	ctx.ResponseError(proto.SUCCEED, "")
}

func (g *Gate) InitHTTP() error {
	g.Router = gmux.NewRouter()

	log.Info0("Register HTTP endpoint \"/v1/namespace\", \"/v1/group\", \"/v1/user\".")
	g.Router.HandleFunc("/v1/{entity:namespace|group|user}", EntityList).Methods("GET")
	g.Router.HandleFunc("/v1/{entity:namespace|group|user}", EntityAlter).Methods("POST", "DELETE")

	log.Info0("Register HTTP endpoint \"/v1/msg\"")
	g.Router.HandleFunc("/v1/msg", PullMessage).Methods("GET")
	g.Router.HandleFunc("/v1/msg", PushMessage).Methods("POST")

	log.Info0("Register HTTP endpoint \"/v1/sub\"")
	g.Router.HandleFunc("/v1/sub", Subscribe).Methods("POST", "DELETE")

	log.Info0("Register HTTP endpoint \"/v1/connect\"")
	g.Router.HandleFunc("/v1/connect", Connect).Methods("POST")

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
