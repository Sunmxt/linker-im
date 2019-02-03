package gate

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/proto"
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

	RPC        *ServiceRPCClient
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
    if err = json.Unmarshal(buf, ireq); err != nil {
        ctx.ResponseError(proto.INVALID_ARGUMENT, err.Error())
        return nil, err
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

func (ctx *APIRequestContext) BeginRPC() (*ServiceRPCClient, error) {
	var err error
	if ctx.EnableTimeout {
		ctx.RPC, err = NewServiceRPCClient(ctx.Timeout)
	} else {
		ctx.RPC, err = TryNewServiceRPCClient()
	}

	return ctx.RPC, err
}

func (ctx *APIRequestContext) EndRPC(err error) {
	if ctx.RPC != nil {
		ctx.RPC.Close(err)
	}
	ctx.RPC = nil
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
	var raw []byte
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
        if err = ctx.WriteJson(proto.HTTPListResponse{
			APIVersion:   1,
			Data:         nil,
			Code:         proto.SERVER_INTERNAL_ERROR,
			ErrorMessage: ctx.CodeMessage,
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
		    APIVersion:   ctx.Version,
			Data:         ctx.Data,
			Code:         ctx.Code,
			ErrorMessage: ctx.CodeMessage,
        }) ; err != nil {
            ctx.ResponseError(proto.SERVER_INTERNAL_ERROR, "JSON marshal failure (" + err.Error() + ").")
        }
	}

	ctx.EndRPC(nil)
}

// API

func EntityAlter(w http.ResponseWriter, req *http.Request) {
}

func NamespaceOperate(w http.ResponseWriter, req *http.Request) {
	var rpcClient *ServiceRPCClient
    var req proto.EntityAlterV1

	ctx, err := NewAPIListRequestContext(w, req, &req)
	if err != nil {
		return
	}
	defer ctx.Finalize()

	rpcClient, err = ctx.BeginRPC()
	if err != nil {
		return
	}

	switch ctx.Req.Method {
	case "POST":
		err = rpcClient.NamespaceAdd(req.Entities)
	case "DELETE":
		err = rpcClient.NamespaceRemove(req.Entities)
	default:
		ctx.StatusCode = 400
		ctx.ResponseError(proto.INVALID_ARGUMENT, "Invalid operation")
		return
	}
	if err != nil {
		authErr, isAuthErr := err.(server.AuthError)
		if isAuthErr {
			ctx.ResponseError(proto.ACCESS_DEINED, authErr.Error())
		} else {
			ctx.ResponseError(proto.SERVER_INTERNAL_ERROR, "(rpc failure) "+err.Error())
		}
		return
	}

	ctx.SetResponse(nil)
}

func ListNamespace(w http.ResponseWriter, req *http.Request) {
	var rpcClient *ServiceRPCClient
	var namespaces []string

	ctx, err := NewAPIListRequestContext(w, req)
	if err != nil {
		return
	}
	defer ctx.Finalize()

	rpcClient, err = ctx.BeginRPC()
	if err != nil {
		return
	}

	namespaces, err = rpcClient.NamespaceList()
	ctx.EndRPC(err)
	if err != nil {
		authErr, isAuthErr := err.(server.AuthError)
		if isAuthErr {
			ctx.ResponseError(proto.ACCESS_DEINED, authErr.Error())
		} else {
			ctx.ResponseError(proto.SERVER_INTERNAL_ERROR, "(rpc failure) "+err.Error())
		}
		return
	} else {
		ctx.ListData = make([]interface{}, 0, len(namespaces))
		for _, ns := range namespaces {
			ctx.ListData = append(ctx.ListData, ns)
		}
	}

	ctx.ResponseError(proto.SUCCEED, "")
}

func GroupOperation(w http.ResponseWriter, req *http.Request) {

}

func ListGroup(w http.ResponseWriter, req *http.Request) {
}

func ListUser(w http.ResponseWriter, req *http.Request) {
}

func UserOperation(w http.ResponseWriter, req *http.Request) {
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

	ctx, err := NewAPIMapRequestContext(w, req)
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

	raw, ok := ctx.Data["enc"]
	if !ok {
		enc = "txt"
	} else {
		enc, ok = raw.(string)
		if !ok || (enc != "txt" && enc != "b64") {
			ctx.ResponseError(proto.INVALID_ARGUMENT, "")
			return
		}
	}

	raw, ok = ctx.Data["usr"]
	if !ok {
		ctx.ResponseError(proto.INVALID_ARGUMENT, "Missing user.")
		return
	}
	usr, ok = raw.(string)
	if !ok || usr == "" {
		ctx.ResponseError(proto.INVALID_ARGUMENT, "")
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
			msg[idx].Raw = base64.StdEncoding.EncodeToString([]byte(msg[idx].Raw))
		}
	}
	for idx := range msg {
		resp = append(resp, msg[idx])
	}
	ctx.SetListResponse(resp)
	ctx.Finalize()
}

func (g *Gate) InitHTTP() error {
	g.Router = gmux.NewRouter()

	log.Info0("Register HTTP endpoint \"/namespace\"")
	g.Router.HandleFunc("/namespace", NamespaceOperate).Methods("POST", "DELETE")
	g.Router.HandleFunc("/namespace", ListNamespace).Methods("GET")
    g.Router.HandleFunc("/namespace", EntityAlter).Methods("POST", "DELETE")

	log.Info0("Register HTTP endpoint \"/group\"")
	g.Router.HandleFunc("/group", ListGroup).Methods("GET")
	g.Router.HandleFunc("/group", GroupOperation).Methods("POST", "DELETE")

	log.Info0("Register HTTP endpoint \"/user\"")
	g.Router.HandleFunc("/user", ListUser).Methods("GET")
	g.Router.HandleFunc("/user", UserOperation).Methods("POST", "DELETE")

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
