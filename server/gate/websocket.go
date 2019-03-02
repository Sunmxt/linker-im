package gate

import (
	"fmt"
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server"
	ws "github.com/gorilla/websocket"
	guuid "github.com/satori/go.uuid"
	"net/http"
)

var upgrader = ws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(req *http.Request) bool {
		return true
	},
}

func wsWriteError(conn *ws.Conn, reqID uint32, msg string, ignoreDebug bool) error {
	return wsWriteProtocolUnit(conn, reqID, &proto.ErrorResponse{
		Err: WrapErrorMessage(msg, guuid.NewV4().String(), ignoreDebug),
	})
}

func wsWriteSuccess(conn *ws.Conn, reqID uint32) error {
	return wsWriteProtocolUnit(conn, reqID, &proto.ErrorResponse{
		Err: "",
	})
}

func wsWriteConnectError(conn *ws.Conn, msg string) error {
	return wsWriteProtocolUnit(conn, 0, &proto.ConnectResultV1{
		AuthError: msg,
		Session:   "",
	})
}

func wsWriteProtocolUnit(conn *ws.Conn, reqID uint32, unit proto.ProtocolUnit) error {
	resp := &proto.Request{
		RequestID: reqID,
		Body:      unit,
	}
	bin := make([]byte, resp.Len())
	err := resp.Marshal(bin)
	if err != nil {
		log.Error("wsWriteProtocolUnit binary Marshal error: " + err.Error())
		return err
	}
	if err = conn.WriteMessage(ws.BinaryMessage, bin); err != nil {
		log.Error("wsWriteProtocolUnit failed when writing to connection: " + err.Error())
		return err
	}
	return nil
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
			wsWriteError(conn, req.RequestID, "(rpc error) "+err.Error(), false)
		} else {
			wsWriteConnectError(conn, "Access denied: "+err.Error())
		}
		conn.Close()
		return
	}
	if err = wsWriteProtocolUnit(conn, req.RequestID, result); err != nil {
		log.Error("WebsocketConnect() failed to write protocol unit: " + err.Error())
		conn.Close()
		return
	}
	go WebsocketServe(conn)
	go WebsocketPush(conn)
}

func WebsocketPush(conn *ws.Conn) {
}

func WebsocketKeepalive(conn *ws.Conn, req *proto.Request) error {
	return wsWriteError(conn, req.RequestID, "Unsupported.", true)
}

func WebsocketServe(conn *ws.Conn) {
WebsocketLoopOp:
	for {
		msgType, dat, err := conn.ReadMessage()
		if err != nil {
			if ws.IsCloseError(err, ws.CloseGoingAway, ws.CloseNormalClosure, ws.CloseNoStatusReceived, ws.CloseAbnormalClosure) {
				conn.Close()
				return
			} else {
				log.Error("Failed to read websocket message: " + err.Error())
			}
			continue
		}
		if msgType != ws.BinaryMessage {
			wsWriteError(conn, 0, "Invalid type of websocket message.", true)
			continue
		}
		req, consume := &proto.Request{}, uint(0)
		consume, err = req.Unmarshal(dat)
		if err != nil {
			wsWriteError(conn, 0, "Invalid protocol unit: "+err.Error(), true)
			continue
		}
		switch req.OpCode {
		case proto.OP_SUB:
			sub := proto.Subscription{}
			bodyConsume, bodyErr := sub.Unmarshal(dat[6:])
			consume += bodyConsume
			if bodyErr != nil {
				wsWriteError(conn, req.RequestID, "Invalid protocol unit: "+bodyErr.Error(), true)
				continue WebsocketLoopOp
			}
			err = gate.subscribe(sub)
			if err == nil {
				wsWriteSuccess(conn, req.RequestID)
			}

		case proto.OP_KEEPALIVE:
			WebsocketKeepalive(conn, req)
			continue

		case proto.OP_PUSH:
			push := proto.MessagePushV1{}
			bodyConsume, bodyErr := push.Unmarshal(dat[6:])
			consume += bodyConsume
			if bodyErr != nil {
				wsWriteError(conn, req.RequestID, "Invalid protocol unit: "+bodyErr.Error(), true)
				continue WebsocketLoopOp
			}
			result, pushErr := gate.push(push.Namespace, push.Session, push.Msgs)
			if err == nil {
				err = wsWriteProtocolUnit(conn, req.RequestID, proto.PushResultList(result))
			} else {
				err = pushErr
			}

		default:
			wsWriteError(conn, req.RequestID, fmt.Sprintf("Unsupported protocol unit: %v", req.OpCode), true)
		}
		if err != nil {
			if authErr, isAuthErr := err.(server.AuthError); !isAuthErr {
				log.Error("RPC Error: " + err.Error())
				wsWriteError(conn, req.RequestID, "(rpc failure) "+err.Error(), false)
			} else {
				wsWriteError(conn, req.RequestID, authErr.Error(), true)
			}
		}
		if consume != uint(len(dat)) {
			log.Warnf("Length of buffer (%v) is not equal to length of consumed bytes (%v).", uint(len(dat)), consume)
		}
	}
}
