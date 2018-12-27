package api

import (
	"io"
	"net/http"
)

func Health(writer http.ResponseWriter, req *http.Request) {
	io.WriteString(writer, "ok")
}
