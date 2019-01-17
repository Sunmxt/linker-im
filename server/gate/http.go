package gate

import (
	"encoding/json"
	"fmt"
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server/resource"
	gmux "github.com/gorilla/mux"
	guuid "github.com/satori/go.uuid"
	"io"
	"net/http"
	"strconv"
	//"github.com/Sunmxt/buger/jsonparser"
)

var APILog *log.Logger

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
	Data        map[string]interface{}
	ListData    []interface{}

	RPC        *ServiceRPCClient
	StatusCode int
	Namespace  string
	Group      string

	Log *log.Logger
}

func NewAPIListRequestContext(w http.ResponseWriter, req *http.Request) (*APIRequestContext, error) {
	var err error

	ctx := NewEmptyAPIRequestContext(w, req)
	defer func() {
		// If any error occurs, shut the context.
		if err != nil {
			ctx.Finalize()
		}
	}()

	buf, ireq := make([]byte, req.ContentLength), proto.HTTPListRequest{}

	// Parse request json.
	readc, err := io.ReadFull(req.Body, buf)
	ctx.Log.DebugLazy(func() string {
		return fmt.Sprintf("Read %v bytes from body.", readc)
	})
	ctx.Log.TraceLazy(func() string {
		return "Request body: " + string(buf)
	})
	if err != nil {
		ctx.Code = proto.INVALID_ARGUMENT
		ctx.CodeMessage = err.Error()
		return nil, err
	}
	err = json.Unmarshal(buf, &ireq)
	if err != nil {
		ctx.Code = proto.INVALID_ARGUMENT
		ctx.CodeMessage = err.Error()
		return nil, err
	}
	ctx.ListData = ireq.Arguments

	if err = ctx.initializeContext(); err != nil {
		return nil, err
	}

	return ctx, nil
}

func NewAPIMapRequestContext(w http.ResponseWriter, req *http.Request) (*APIRequestContext, error) {
	var err error

	ctx := NewEmptyAPIRequestContext(w, req)
	defer func() {
		// If any error occurs, shut the context.
		if err != nil {
			ctx.Finalize()
		}
	}()

	buf, ireq := make([]byte, req.ContentLength), proto.HTTPMapRequest{}

	// Parse request json.
	readc, err := io.ReadFull(req.Body, buf)
	ctx.Log.DebugLazy(func() string {
		return fmt.Sprintf("Read %v bytes from body.", readc)
	})
	ctx.Log.TraceLazy(func() string {
		return "Request body: " + string(buf)
	})
	if err != nil {
		ctx.Code = proto.INVALID_ARGUMENT
		ctx.CodeMessage = err.Error()
		return nil, err
	}
	err = json.Unmarshal(buf, &ireq)
	if err != nil {
		ctx.Code = proto.INVALID_ARGUMENT
		ctx.CodeMessage = err.Error()
		return nil, err
	}
	ctx.Data = ireq.Arguments

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

	return nil
}

func (ctx *APIRequestContext) ParseTimeout() error {
	timeouts, ok := ctx.Req.Form["timeout"]
	if ok && len(timeouts) > 0 {
		timeout, err := strconv.ParseUint(timeouts[0], 10, 32)
		if err != nil {
			ctx.Code = proto.INVALID_ARGUMENT
			ctx.CodeMessage = err.Error()
			ctx.Finalize()
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

func (ctx *APIRequestContext) ResponseWithList(list []interface{}) {
	ctx.Data = nil
	ctx.ListData = list
}

func (ctx *APIRequestContext) ResponseWithMap(mapping map[string]interface{}) {
	ctx.Data = mapping
	ctx.ListData = nil
}

func (ctx *APIRequestContext) Finalize() {
	var raw []byte
	var err error

	ctx.Writer.WriteHeader(ctx.StatusCode)

	switch ctx.Code {
	case proto.SERVER_INTERNAL_ERROR:
		// Log error
		ctx.Log.Error(ctx.CodeMessage)

		if !Config.DebugMode.Value {
			// Mask error message.
			ctx.CodeMessage = "Server raise an exception with ID \"" + ctx.RequestID.String() + "\""
		} else {
			// Add Request ID to error message
			ctx.CodeMessage = ctx.CodeMessage + "[ID = " + ctx.RequestID.String() + "]"
		}

		// Try to return error with API Format.
		raw, err = json.Marshal(proto.HTTPListResponse{
			APIVersion:   1,
			Data:         nil,
			Code:         proto.SERVER_INTERNAL_ERROR,
			ErrorMessage: ctx.CodeMessage,
		})

		// Fallback to HTTP 500
		if err != nil {
			http.Error(ctx.Writer, ctx.CodeMessage, 500)
		} else {
			_, err = ctx.Writer.Write(raw)
		}
	default:
		if ctx.CodeMessage == "" {
			// Set default message.
			ctx.CodeMessage = proto.ErrorCodeText(ctx.Code)
		}
		if ctx.Data != nil {
			raw, err = json.Marshal(proto.HTTPMapResponse{
				APIVersion:   ctx.Version,
				Data:         ctx.Data,
				Code:         ctx.Code,
				ErrorMessage: ctx.CodeMessage,
			})
		} else {
			raw, err = json.Marshal(proto.HTTPListResponse{
				APIVersion:   ctx.Version,
				Data:         ctx.ListData,
				Code:         ctx.Code,
				ErrorMessage: ctx.CodeMessage,
			})
		}
		if err != nil {
			// Fallback to internal error.
			ctx.Code = proto.SERVER_INTERNAL_ERROR
			ctx.CodeMessage = "JSON marshal failure (" + err.Error() + ")."
			ctx.Finalize()
		} else {
			_, err = ctx.Writer.Write(raw)
		}
	}

	ctx.EndRPC(nil)
}

// API
func Health(writer http.ResponseWriter, req *http.Request) {
	io.WriteString(writer, "ok")
}

func ListResource(w http.ResponseWriter, req *http.Request) {
	res := resource.Registry.ListResources()
	data := make([]interface{}, 0, len(res))
	for _, r := range res {
		data = append(data, r)
	}

	bin, err := json.Marshal(proto.HTTPListResponse{
		APIVersion:   1,
		Data:         data,
		Code:         0,
		ErrorMessage: proto.ErrorMessageFromCode[0],
	})

	if err != nil {
		APILog.Errorf("Json marshal failure: %v", err.Error())
	}
	w.Write(bin)
}

func NamespaceOperate(w http.ResponseWriter, req *http.Request) {
	var rpcClient *ServiceRPCClient

	ctx, err := NewAPIListRequestContext(w, req)
	if err != nil {
		return
	}
	defer ctx.Finalize()

	namespaces := make([]string, 0, len(ctx.ListData))
	for _, raw := range ctx.ListData {
		ns, ok := raw.(string)
		if !ok {
			ctx.Code = proto.INVALID_ARGUMENT
			ctx.CodeMessage = "Invalid arguments."
			return
		}
		namespaces = append(namespaces, ns)
	}

	rpcClient, err = ctx.BeginRPC()
	if err != nil {
		return
	}

	switch ctx.Req.Method {
	case "POST":
		err = rpcClient.NamespaceAdd(namespaces)
	case "DELETE":
		err = rpcClient.NamespaceRemove(namespaces)
	default:
		ctx.Code = proto.INVALID_ARGUMENT
		ctx.CodeMessage = "Invalid opertaion."
		ctx.StatusCode = 400
		return
	}
	if err != nil {
		authErr, isAuthErr := err.(resource.ResourceAuthError)
		if isAuthErr {
			ctx.Code = proto.ACCESS_DEINED
			ctx.CodeMessage = authErr.Error()
		} else {
			ctx.Code = proto.SERVER_INTERNAL_ERROR
			ctx.CodeMessage = "(rpc return failure) " + err.Error()
		}
		return
	}

	ctx.ResponseWithList(nil)
	ctx.Code = proto.SUCCEED
	ctx.CodeMessage = ""
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
		authErr, isAuthErr := err.(resource.ResourceAuthError)
		if isAuthErr {
			ctx.Code = proto.ACCESS_DEINED
			ctx.CodeMessage = authErr.Error()
		} else {
			ctx.Code = proto.SERVER_INTERNAL_ERROR
			ctx.CodeMessage = "(rpc return failure) " + err.Error()
		}
		return
	} else {
		ctx.ListData = make([]interface{}, 0, len(namespaces))
		for _, ns := range namespaces {
			ctx.ListData = append(ctx.ListData, ns)
		}
	}

	ctx.Code = proto.SUCCEED
	ctx.CodeMessage = ""
}

func GroupOperation(w http.ResponseWriter, req *http.Request) {

}

func ListGroup(w http.ResponseWriter, req *http.Request) {
}

func ListUser(w http.ResponseWriter, req *http.Request) {
}

func UserOperation(w http.ResponseWriter, req *http.Request) {
}

func RegisterHTTPAPI(mux *gmux.Router) error {
	// Healthz
	APILog.Info0("Register HTTP health check at \"/healthz\"")
	mux.HandleFunc("/healthz", Health)

	// Resource list.
	APILog.Info0("Register HTTP Resource listing at \"/resources\"")
	mux.HandleFunc("/resources", ListResource)

	// Namespace
	APILog.Info0("Register HTTP endpoint \"/namespace\"")
	mux.HandleFunc("/namespace", NamespaceOperate).Methods("POST", "DELETE")
	mux.HandleFunc("/namespace", ListNamespace).Methods("GET")
	APILog.Info0("Register HTTP endpoint \"/group\"")
	mux.HandleFunc("/group", ListGroup).Methods("GET")
	mux.HandleFunc("/group", GroupOperation).Methods("POST", "DELETE")
	APILog.Info0("Register HTTP endpoint \"/user\"")
	mux.HandleFunc("/user", ListUser).Methods("GET")
	mux.HandleFunc("/user", UserOperation).Methods("POST", "DELETE")
	//APILog.Info0("Register HTTP endpoint \"msg\"")

	return nil
}

func NewHTTPAPIMux() (http.Handler, error) {
	mux := gmux.NewRouter()
	if err := RegisterHTTPAPI(mux); err != nil {
		return nil, err
	}

	return mux, nil
}

func init() {
	APILog = log.NewLogger()
}
