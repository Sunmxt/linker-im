package proto

import (
    guuid "github.com/satori/go.uuid"
    "fmt"
)

const (
    SUCCEED = 0
    INVALID_ARGUMENT = 1
)

var ErrorMessageFromCode map[uint32]string = map[uint32]string{
    SUCCEED: "succeed.",
}

func ErrorCodeText(code uint32) string {
    err, ok := ErrorMessageFromCode[code]
    if !ok {
        return fmt.Sprintf("unknown error (code = %v)", code)
    }
    return err
}

type HTTPMapResponse struct {
    APIVersion      uint32              `json:"ver"`
    Data            map[string]interface{}   `json:"data"`
    Code            uint32              `json:"code"`
    ErrorMessage    string              `json:"msg"`
}

type HTTPListResponse struct {
    APIVersion      uint32              `json:"ver"`
    Data            []interface{}            `json:"data"`
    Code            uint32              `json:"code"`
    ErrorMessage    string              `json:"msg"`
}

type HTTPListRequest struct {
    APIVersion      uint32              `json:"ver"`
    Arguments       []interface{}       `json:"args"`
    RequestID       guuid.UUID          `json:"-"`
}

type HTTPMapRequest struct {
    APIVersion      uint32              `json:"ver"`
    Arguments       map[string]interface{}       `json:"args"`
    RequestID       guuid.UUID          `json:"-"`
}

func (req *HTTPListResponse) Identifier() string {
    return req.RequestID.String()
}

func (req *HTTPMapResponse) Identifier() string {
    return req.RequestID.String()
}
