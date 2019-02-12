package proto

import (
	"fmt"
)

const (
	SUCCEED               = uint32(0)
	INVALID_ARGUMENT      = uint32(1)
	TIMEOUT               = uint32(2)
	ACCESS_DEINED         = uint32(3)
	SERVER_INTERNAL_ERROR = uint32(4)
)

var ErrorMessageFromCode map[uint32]string = map[uint32]string{
	SUCCEED: "succeed.",
	TIMEOUT: "Request timeout.",
}

func ErrorCodeText(code uint32) string {
	err, ok := ErrorMessageFromCode[code]
	if !ok {
		return fmt.Sprintf("unknown error (code = %v)", code)
	}
	return err
}

type HTTPResponse struct {
	Version uint32      `json:"ver"`
	Data    interface{} `json:"data"`
	Code    uint32      `json:"code"`
	Msg     string      `json:"msg"`
}

type EntityAlterV1 struct {
	Entities []string `json:"args"`
}

type MessagePushV1 struct {
	Msgs []MessageBody `json:"msg"`
}
