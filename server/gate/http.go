package gate

import (
	"encoding/json"
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server/resource"
	"io"
	"net/http"
	//"github.com/Sunmxt/buger/jsonparser"
)

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
		log.Errorf("Json marshal failure: %v", err.Error())
	}
	w.Write(bin)
}

func RegisterHTTPAPI(mux *http.ServeMux) error {
	// Healthz
	log.Info0("Register HTTP health check at \"/healthz\"")
	mux.HandleFunc("/healthz", Health)

	// Resource list.
	log.Info0("Register HTTP Resource listing at \"/resources\"")
	mux.HandleFunc("/resources", ListResource)
	return nil
}

func NewHTTPAPIMux() (*http.ServeMux, error) {
	mux := http.NewServeMux()
	if err := RegisterHTTPAPI(mux); err != nil {
		return nil, err
	}

	return mux, nil
}
