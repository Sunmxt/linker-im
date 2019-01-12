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
	//"github.com/Sunmxt/buger/jsonparser"
)

var APILog *log.Logger

type IdentifiedInstance interface {
	Identifier() string
}

// Extract json request with argument list from HTTP body.
func ParseJSONAPIListRequest(req *http.Request) (*proto.HTTPListRequest, error) {
	buf, ireq := make([]byte, 0, req.ContentLength), proto.HTTPListRequest{
		RequestID: guuid.NewV4(),
	}

	err := json.Unmarshal(buf, ireq)

	return &ireq, err
}

// Extract json request with mapping-type arguments from HTTP body.
func ParseJSONAPIMapRequest(req *http.Request) (*proto.HTTPMapRequest, error) {
	buf, ireq := make([]byte, 0, req.ContentLength), proto.HTTPMapRequest{
		RequestID: guuid.NewV4(),
	}

	err := json.Unmarshal(buf, ireq)

	return &ireq, err
}

func InternalError(err error, w http.ResponseWriter, req IdentifiedInstance) {
	errmsg := err.Error()

	APILog.ErrorMap(map[string]interface{}{
		"request-id": req.Identifier(),
	}, err)

	if !Config.DebugMode.Value {
		errmsg = "Server raise an exception with ID \"" + req.Identifier() + "\""
	} else {
		errmsg += "(ID = " + req.Identifier() + " )."
	}

	http.Error(w, errmsg, 500)
}

// Response with list
func ResponseList(version uint32, w http.ResponseWriter, req IdentifiedInstance, data []interface{}, getErr func() (string, uint32)) {
	var msg string
	var code uint32

	if getErr != nil {
		msg, code = getErr()
	} else {
		msg, code = "", 0
	}

	raw, err := json.Marshal(proto.HTTPListResponse{
		APIVersion:   version,
		Data:         data,
		Code:         code,
		ErrorMessage: msg,
	})
	if err != nil {
		InternalError(fmt.Errorf("Json marshal failure \"%v\"", err.Error()), w, req)
		return
	}

	_, err = w.Write(raw)
	if err != nil {
		InternalError(err, w, req)
	}
}

// Response with list and version 1
func ResponseListV1(w http.ResponseWriter, req IdentifiedInstance, data []interface{}, getErr func() (string, uint32)) {
	ResponseList(1, w, req, data, getErr)
}

// Response with empty list and version 1
func ResponseEmptyListV1(w http.ResponseWriter, req IdentifiedInstance, getErr func() (string, uint32)) {
	ResponseList(1, w, req, nil, getErr)
}

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

func NewNamespace(w http.ResponseWriter, req *http.Request) {
	//ireq, err := ParseJSONAPIListRequest(req)
	//namespaces := make([]string, 0, len(ireq.Arguments))
	//for _, raw := range ireq.Arguments {
	//	ns, ok := raw.(string)
	//	if !ok {
	//		ResponseEmptyListV1(w, ireq, func() (string, uint32) {
	//			return "invalid arguments.", proto.INVALID_ARGUMENT
	//		})
	//	}
	//}
	// RPC here
}

func ListNamespace(w http.ResponseWriter, req *http.Request) {

}

func RegisterHTTPAPI(mux *gmux.Router) error {
	// Healthz
	APILog.Info0("Register HTTP health check at \"/healthz\"")
	mux.HandleFunc("/healthz", Health)

	// Resource list.
	APILog.Info0("Register HTTP Resource listing at \"/resources\"")
	mux.HandleFunc("/resources", ListResource)

	// Namespace
	APILog.Info0("Resource HTTP Namespace at \"/namespace\"")
	mux.HandleFunc("/namespace/{name}", NewNamespace).Methods("POST")
	mux.HandleFunc("/namespace", ListNamespace)
	//mux.HandleFunc("/namespace/{name}", NamespaceFunc).Methods("GET")
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
	APILog.Fields["entity"] = "http-api"
}
