package svc

import (
	"io"
	"net/http"
)

func Healthz(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "ok")
}
