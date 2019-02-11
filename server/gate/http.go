package gate

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server"
	sc "github.com/Sunmxt/linker-im/server/svc/client"
	gmux "github.com/gorilla/mux"
	guuid "github.com/satori/go.uuid"
	"io"
	"net/http"
	"strconv"
	//"github.com/Sunmxt/buger/jsonparser"
)

// API Request context
type APIRequestContext struct {
	EnableTimeout bool
	Timeout       uint32

	Writer    http.ResponseWriter
	Req       *http.Request
	RequestID guuid.UUID

	Version     uint32
	Code        uint32
	CodeMessage string
	Data        interface{}

	RPC        *sc.ServiceClient
	node       *ServiceNode
	StatusCode int
	Namespace  string
	Group      string

	Log *log.Logger
}

func NewRequestContext(w http.ResponseWriter, req *http.Request, ireq interface{}) (*APIRequestContext, error) {
	ctx, buf := NewEmptyAPIRequestContext(w, req), make([]byte, req.ContentLength)

	readc, err := io.ReadFull(req.Body, buf)
	ctx.Log.DebugLazy(func() string {
		return fmt.Sprintf("Read %v bytes from body.", readc)
	})
	ctx.Log.TraceLazy(func() string {
		return "Request body: " + string(buf)
	})
	if err != nil {
		ctx.ResponseError(proto.INVALID_ARGUMENT, err.Error())
		return nil, err
	}
	if ireq != nil {
		if err = json.Unmarshal(buf, ireq); err != nil {
			ctx.ResponseError(proto.INVALID_ARGUMENT, err.Error())
			return nil, err
		}
	}
	if err = ctx.initializeContext(); err != nil {
		return nil, err
	}
	return ctx, nil
}

func NewEmptyAPIRequestContext(w http.ResponseWriter, req *http.Request) *APIRequestContext {
	ctx := &APIRequestContext{
		Writer:     w,
		Req:        req,
		RequestID:  guuid.NewV4(),
		Log:        log.NewLogger(),
		StatusCode: 200,
	}
	ctx.Log.Fields["entity"] = "http"
	ctx.Log.Fields["request"] = ctx.RequestID.String()
	return ctx
}

func (ctx *APIRequestContext) initializeContext() error {
	if err := ctx.Req.ParseForm(); err != nil {
		return err
	}

	if err := ctx.ParseTimeout(); err != nil {
		return err
	}

	ctx.ParseNamespace()

	return nil
}

func (ctx *APIRequestContext) ParseTimeout() error {
	timeouts, ok := ctx.Req.Form["timeout"]
	if ok && len(timeouts) > 0 {
		timeout, err := strconv.ParseUint(timeouts[0], 10, 32)
		if err != nil {
			ctx.ResponseError(proto.INVALID_ARGUMENT, err.Error())
			return err
		}
		ctx.EnableTimeout = true
		ctx.Timeout = uint32(timeout)

	} else {
		ctx.EnableTimeout = false
		ctx.Timeout = 0
	}
	return nil
}

func (ctx *APIRequestContext) BeginRPC() (*sc.ServiceClient, error) {
	var err error
	ctx.node, err = gate.LB.RoundRobinSelect()
	if err != nil {
		return nil, err
	}
	if ctx.EnableTimeout {
		ctx.RPC, err = ctx.node.Connect(ctx.Timeout)
	} else {
		ctx.RPC, err = ctx.node.TryConnect()
	}
	return ctx.RPC, err
}

func (ctx *APIRequestContext) EndRPC(err error) {
	if ctx.RPC != nil && ctx.node != nil {
		ctx.node.Disconnect(ctx.RPC, err)
	}
	ctx.RPC = nil
	ctx.node = nil
}

func (ctx *APIRequestContext) ParseNamespace() {
	raw, ok := ctx.Req.Form["ns"]
	if ok && len(raw) > 0 {
		ctx.Namespace = raw[0]
	} else {
		ctx.Namespace = ""
	}
}

func (ctx *APIRequestContext) WriteJson(resp interface{}) error {
	raw, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	if _, err = ctx.Writer.Write(raw); err != nil {
		return err
	}
	return nil
}

func (ctx *APIRequestContext) ResponseError(code uint32, msg string) {
	ctx.Code = code
	ctx.CodeMessage = msg
	ctx.Finalize()
}

func (ctx *APIRequestContext) Finalize() {
	var err error

	ctx.Writer.WriteHeader(ctx.StatusCode)

	if ctx.Code == proto.SERVER_INTERNAL_ERROR {
		// Log error
		ctx.Log.Error(ctx.CodeMessage)

		if !gate.config.DebugMode.Value {
			// Mask error message.
			ctx.CodeMessage = "Server raise an exception with ID \"" + ctx.RequestID.String() + "\""
		} else {
			// Add Request ID to error message
			ctx.CodeMessage = ctx.CodeMessage + "[ID = " + ctx.RequestID.String() + "]"
		}

		// Try to return error with API Format.
		if err = ctx.WriteJson(proto.HTTPResponse{
			Version: 1,
			Data:    nil,
			Code:    proto.SERVER_INTERNAL_ERROR,
			Msg:     ctx.CodeMessage,
		}); err != nil {
			// Fallback to HTTP 500
			http.Error(ctx.Writer, ctx.CodeMessage, 500)
		}
	} else {
		if ctx.CodeMessage == "" {
			// Set default message.
			ctx.CodeMessage = proto.ErrorCodeText(ctx.Code)
		}
		if err = ctx.WriteJson(proto.HTTPResponse{
			Version: ctx.Version,
			Data:    ctx.Data,
			Code:    ctx.Code,
			Msg:     ctx.CodeMessage,
		}); err != nil {
			ctx.ResponseError(proto.SERVER_INTERNAL_ERROR, "JSON marshal failure ("+err.Error()+").")
		}
	}

	ctx.EndRPC(nil)
}

// API

func EntityList(w http.ResponseWriter, req *http.Request) {
	var client *sc.ServiceClient
	var rpcErr error
	vars := gmux.Vars(req)
	ctx, err := NewRequestContext(w, req, nil)
	if err != nil {
		return
	}
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

	raw, ok := ctx.Req.Form["enc"]
	if !ok {
		enc = "txt"
	}
	if len(raw) > 0 {
		enc = raw[0]
	}
	if enc != "txt" && enc != "b64" {
		ctx.ResponseError(proto.INVALID_ARGUMENT, "Invalid enc \""+enc+"\"")
		return
	}

	raw, ok = ctx.Req.Form["u"]
	if !ok {
		ctx.ResponseError(proto.INVALID_ARGUMENT, "Missing u.")
		return
	}
	if len(raw) > 0 {
		usr = raw[0]
	}
	if usr == "" {
		ctx.ResponseError(proto.INVALID_ARGUMENT, "Empty u.")
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
	g.Router.HandleFunc("/v1/{entity}", EntityList).Methods("GET")
	g.Router.HandleFunc("/v1/{entity}", EntityAlter).Methods("POST", "DELETE")

	log.Info0("Register HTTP endpoint \"msg\"")
	g.Router.HandleFunc("/msg", PullMessage).Methods("GET")

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
