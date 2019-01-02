package proto

const (
    SUCCEED = 0
)

var ErrorMessageFromCode map[uint32]string = map[uint32]string{
    SUCCEED: "succeed.",
}

type HTTPMapResponse struct {
    APIVersion      uint32              `json:"ver"`
    Data            map[string]string   `json:"data"`
    Code            uint32              `json:"code"`
    ErrorMessage    string              `json:"msg"`
}

type HTTPListResponse struct {
    APIVersion      uint32              `json:"ver"`
    Data            []string            `json:"data"`
    Code            uint32              `json:"code"`
    ErrorMessage    string              `json:"msg"`
}

type HTTPListRequest struct {
    APIVersion      uint32              `json:"ver"`
    Arguments       []interface{}       `json:"args"`
}

type HTTPMapRequest struct {
    APIVersion      uint32              `json:"ver"`
    Arguments       map[string]interface{}       `json:"args"`
}
