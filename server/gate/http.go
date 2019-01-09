package gate

import (
	"encoding/json"
	"fmt"
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server/resource"
	gmux "github.com/gorilla/mux"
	"io"
	"net/http"
	//"github.com/Sunmxt/buger/jsonparser"
)

var APILog *log.Logger

//func ServeMethodMux struct {
//    Handlers map[string]http.Handler
//}
//
//func NewServeMethodMux() *ServeMethodMux {
//}
//
//func (mux *ServeMethodMux) Handle(patten string, handler Handler)

func Health(writer http.ResponseWriter, req *http.Request) {
	io.WriteString(writer, "ok")
}

func ListResource(w http.ResponseWriter, req *http.Request) {
	bin, err := json.Marshal(proto.HTTPListResponse{
		APIVersion:   1,
		Data:         resource.Registry.ListResources(),
		Code:         0,
		ErrorMessage: proto.ErrorMessageFromCode[0],
	})

	if err != nil {
		APILog.Errorf("Json marshal failure: %v", err.Error())
	}
	w.Write(bin)
}

func NamespaceFunc(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		fmt.Println("%v", req)
		fmt.Println("%v", gmux.Vars(req))
	default:
		http.Error(w, "Unsupported method.", 400)
	}
}

func NewNamespace(w http.ResponseWriter, req *http.Request) {
	fmt.Println("c")
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
	mux.HandleFunc("/namespace/{name}", NamespaceFunc).Methods("GET")
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
