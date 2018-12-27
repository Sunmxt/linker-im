package log

import (
    "net/http"
    "fmt"
    "math/rand"
)

// Constants
const DEFAULT_LOG_BODY_BUFFER_SIZE = 64

// ProxyResponseWriter
type ProxyResponseWriter struct {
    Origin              http.ResponseWriter
    HookWrite           func ([]byte) (int, error)
    HookWriteHeader     func (int)
}

func (w *ProxyResponseWriter) Header() http.Header {
    return w.Origin.Header()
}

func (w *ProxyResponseWriter) Write(raw []byte) (int, error) {
    if w.HookWrite != nil {
        return w.HookWrite(raw)
    }
    return w.Origin.Write(raw)
}

func (w *ProxyResponseWriter) WriteHeader(statusCode int) {
    if w.HookWriteHeader != nil {
        w.HookWriteHeader(statusCode)
        return
    }
    w.Origin.WriteHeader(statusCode)
}

// LoggedHandler automaticalliy log http response/request.
type LoggedHandler struct {
    Tags                    map[string]interface{}
    OriginFunc              http.Handler
}

// Decorate and attach log system to HandlerFunc.
// Return new HandlerFunc
func LogHandler(handlerFunc http.Handler) *LoggedHandler {
    return TagLogHandler(handlerFunc, map[string]interface{}{})
}

// Decorate and attach log system to HandlerFunc.
// Return new HandlerFunc
// Tags specified will be appended to log line.
func TagLogHandler(handlerFunc http.Handler, tags map[string]interface{}) *LoggedHandler {
    return &LoggedHandler{
        Tags:                   tags,
        OriginFunc:             handlerFunc,
    }
}

func (fun *LoggedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    var statusCode int = 200
    var bodySize uint64 = 0

    proxyWriteHeader := func (code int) {
        statusCode = code
        w.WriteHeader(code)
    }
    proxy := &ProxyResponseWriter{
        Origin:             w,
        HookWriteHeader:    proxyWriteHeader,
        HookWrite:          func (raw []byte) (int, error) {
            written, err := w.Write(raw)
            bodySize += uint64(written)
            return written, err
        },
    }

    fun.OriginFunc.ServeHTTP(proxy, r)

    sid := rand.Uint32()
    // Log briefly
    userAgent, _ := r.Header["User-Agent"]
    InfoMap(fun.Tags, fmt.Sprintf("(%x)[%v] %v %v %v %v %v %v", sid, r.RemoteAddr ,r.Method, r.RequestURI, statusCode, r.ContentLength, bodySize, userAgent[0]))

    // Log Header (Debug level only)
    if GlobalLogLevel() > 3 {
        logHeader := func (header http.Header, leadMsg string) {
            DebugMap(fun.Tags, fmt.Sprintf("(%x) %v", sid, leadMsg))
            for k, v := range header {
                switch len(v) {
                case 0:
                    DebugMap(fun.Tags, fmt.Sprintf("(%x) %v:", sid, k))
                case 1:
                    DebugMap(fun.Tags, fmt.Sprintf("(%x) %v: %v", sid, k, v[0]))
                default:
                    DebugMap(fun.Tags, fmt.Sprintf("(%x) %v: %v", sid, k, v))
                }
            }
        }
        logHeader(r.Header, "--- Request Header ---")
        logHeader(w.Header(), "--- Response Header ---")
    }

}
