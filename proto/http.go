package proto

import (
	"fmt"
)

const (
	SUCCEED               = 0
	INVALID_ARGUMENT      = 1
	TIMEOUT               = 2
	ACCESS_DEINED         = 3
	SERVER_INTERNAL_ERROR = 4
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

type HTTPMapResponse struct {
	APIVersion   uint32                 `json:"ver"`
	Data         map[string]interface{} `json:"data"`
	Code         uint32                 `json:"code"`
	ErrorMessage string                 `json:"msg"`
}

type HTTPListResponse struct {
	APIVersion   uint32        `json:"ver"`
	Data         []interface{} `json:"data"`
	Code         uint32        `json:"code"`
	ErrorMessage string        `json:"msg"`
}

type HTTPListRequest struct {
	APIVersion uint32        `json:"ver"`
	Arguments  []interface{} `json:"args"`
}

type HTTPMapRequest struct {
	APIVersion uint32                 `json:"ver"`
	Arguments  map[string]interface{} `json:"args"`
}
