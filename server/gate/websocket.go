package gate

import (
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server"
	ws "github.com/gorilla/websocket"
	"net/http"
)

var upgrader = ws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func wsWriteError(conn *ws.Conn, err string) {
}

func wsWriteProtocolUnit(conn *ws.Conn, reqID uint32, unit proto.ProtocolUnit) {
}

func wsWriteConnectError(conn *ws.Conn, msg string) {
	resp := &proto.Request{
		Body: &proto.ConnectResultV1{
			AuthError: msg,
			Session:   "",
		},
	}
	bin := make([]byte, resp.Len())
	err := resp.Marshal(bin)
	if err != nil {
		log.Error("wsWriteConnectError binary Marshal error: " + err.Error())
		return
	}
	if err = conn.WriteMessage(ws.BinaryMessage, bin); err != nil {
		log.Error("wsWriteConnectError failed when writing to connection: " + err.Error())
	}
}

func WebsocketConnect(w http.ResponseWriter, r *http.Request) {
	var (
		msgType int
		dat     []byte
		result  *proto.ConnectResultV1
	)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("Websocket upgrade failure: " + err.Error())
		return
	}
	// Connect
	if msgType, dat, err = conn.ReadMessage(); err != nil {
		log.Error("Websocket client connect failure: " + err.Error())
		conn.Close()
		return
	}
	if msgType != ws.BinaryMessage {
		wsWriteConnectError(conn, "Invalid type of websocket messaage.")
		conn.Close()
		return
	}
	connReq := &proto.ConnectV1{}
	req := &proto.Request{Body: connReq}
	if _, err = req.Unmarshal(dat); err != nil {
		wsWriteConnectError(conn, "Invalid connect request.")
		log.Info2("Reject connection for invalid request: " + err.Error())
		conn.Close()
		return
	}
	if result, err = gate.connect(connReq); err != nil {
		if !server.IsAuthError(err) {
			log.Error("RPC Error: " + err.Error())
			wsWriteError(conn, "(rpc error) "+err.Error())
		} else {
			wsWriteConnectError(conn, "Access denied: "+err.Error())
		}
		conn.Close()
		return
	}
	wsWriteProtocolUnit(conn, req.RequestID, result)
	go WebsocketServe(conn)
}

func WebsocketServe(conn *ws.Conn) {
	log.Warn("WebsocketServe not implemented.")
	conn.Close()
}
