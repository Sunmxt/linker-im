package gate

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server"
	sc "github.com/Sunmxt/linker-im/server/svc/client"
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
	node       *server.RPCNode
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
	var rawRPC *server.RPCClient
	ctx.node, err = gate.LB.RoundRobinSelect()
	if err != nil {
		return nil, err
	}
	if ctx.EnableTimeout {
		rawRPC, err = ctx.node.Connect(ctx.Timeout)
	} else {
		rawRPC, err = ctx.node.TryConnect()
	}
	ctx.RPC = (*sc.ServiceClient)(rawRPC)
	return ctx.RPC, err
}

func (ctx *APIRequestContext) ParseAndGetMessagingClientTuple() (string, string, error) {
	var enc string
	raw, ok := ctx.Req.Form["enc"]
	if !ok || len(raw) < 1 {
		return "", "", nil
	}
	if raw[0] != "txt" && raw[0] != "b64" {
		msg := "Invalid enc \"" + enc + "\""
		ctx.ResponseError(proto.INVALID_ARGUMENT, msg)
		return "", "", errors.New(msg)
	}
	enc = raw[0]
	raw, ok = ctx.Req.Form["s"]
	if !ok || len(raw) < 1 {
		msg := "Session missing."
		ctx.ResponseError(proto.INVALID_ARGUMENT, msg)
		return "", "", errors.New(msg)
	}
	if raw[0] == "" {
		msg := "Empty session."
		ctx.ResponseError(proto.INVALID_ARGUMENT, msg)
		return "", "", errors.New(msg)
	}
	return enc, raw[0], nil
}

func (ctx *APIRequestContext) EndRPC(err error) {
	if ctx.RPC != nil && ctx.node != nil {
		ctx.node.Disconnect((*server.RPCClient)(ctx.RPC), err)
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
