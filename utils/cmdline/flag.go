package cmdline

import (
	"fmt"
	"strconv"
	"strings"
)

// UintValue
type UintValue struct {
	Value     uint
	IsDefault bool
	Error     error
	Base      int
}

func NewUintValueDefault(default_value uint) *UintValue {
	return &UintValue{
		Value:     default_value,
		Base:      10,
		IsDefault: true,
		Error:     nil,
	}
}

func NewUintValue() *UintValue {
	return NewUintValueDefault(0)
}

func (val *UintValue) Set(raw string) error {
	actual, err := strconv.ParseUint(raw, val.Base, 32)
	if err != nil {
		return err
	}
	val.IsDefault = false
	val.Value = uint(actual)
	return nil
}

func (val *UintValue) String() string {
	base := val.Base
	if base == 0 {
		base = 10
	}
	return strconv.FormatUint(uint64(val.Value), base)
}

// StringValue
type StringValue struct {
	Value     string
	IsDefault bool
	Error     error
}

func NewStringValueDefault(default_value string) *StringValue {
	return &StringValue{
		Value:     default_value,
		IsDefault: true,
		Error:     nil,
	}
}

func NewStringValue() *StringValue {
	return NewStringValueDefault("")
}

func (val *StringValue) Set(raw string) error {
	val.Value = raw
	val.IsDefault = false
	return nil
}

func (val *StringValue) String() string {
	return val.Value
}

// NetEndpointValue
type NetEndpointValue struct {
	Scheme       string
	UserInfo     string
	Host         string
	Port         uint32
	HasPort      bool
	IsDefault    bool
	Error        error
	ValidSchemes []string
}

func (val *NetEndpointValue) IsSchemeValid(scheme string) bool {
	for _, valid_scheme := range val.ValidSchemes {
		if scheme == valid_scheme {
			return true
		}
	}
	return false
}

func (val *NetEndpointValue) SetAuthority(authority string) error {
	var user_info, host, port string
	if authority == "" {
		return nil
	}

	if -1 != strings.Index(authority, "/") {
		return fmt.Errorf("Invalid charactor \"/\"")
	}

	user_info_splitter_idx := strings.Index(authority, "@")
	if user_info_splitter_idx != -1 {
		user_info = authority[:user_info_splitter_idx]
	}

	port_splitter_idx := strings.Index(authority[user_info_splitter_idx+1:], ":")
	has_port := false
	if port_splitter_idx != -1 {
		has_port = true
		port = authority[port_splitter_idx+1:]
		host = authority[user_info_splitter_idx+1 : port_splitter_idx]
	} else {
		port = "0"
		host = authority[user_info_splitter_idx+1:]
	}

	act_port, err := strconv.ParseUint(port, 10, 32)
	if err != nil {
		return fmt.Errorf("Port should be an integer: %s", port)
	}
	val.Port = uint32(act_port)
	val.UserInfo = user_info
	val.Host = host
	val.HasPort = has_port
	return nil
}

func NewNetEndpointValueDefault(validSchemes []string, netEndpoint string) (*NetEndpointValue, error) {
	new_instance := &NetEndpointValue{
		ValidSchemes: validSchemes,
	}
	err := new_instance.Set(netEndpoint)
	if err != nil {
		return nil, err
	}
	new_instance.IsDefault = true
	return new_instance, nil
}

func NewNetEndpointValue(validSchemes []string) (*NetEndpointValue, error) {
	return NewNetEndpointValueDefault(validSchemes, "")
}

func (val *NetEndpointValue) Set(raw string) error {
	var scheme, authority string
	var err error

	if raw == "" {
		// Allow to be empty.
		scheme = ""
		authority = ""
	} else {
		idx_colon := strings.Index(raw, "://")
		// Determine scheme
		if idx_colon != -1 {
			scheme = raw[:idx_colon]
			authority = raw[idx_colon+3:]
			if !val.IsSchemeValid(scheme) {
				val.Error = fmt.Errorf("Unsupported network endpoint scheme: %v", scheme)
				return val.Error
			}
		} else {
			scheme = ""
			authority = raw
		}

		// Parse authority part.
		if err = val.SetAuthority(authority); err != nil {
			val.Error = fmt.Errorf("Invalid authority format: %v", err.Error())
			return val.Error
		}
	}
	val.Scheme = scheme
	val.IsDefault = false
	return nil
}

func (val *NetEndpointValue) String() string {
	scheme_raw := ""

	if val.Scheme != "" {
		scheme_raw = val.Scheme + "://"
	}
	return scheme_raw + val.AuthorityString()
}

func (val *NetEndpointValue) AuthorityString() string {
	userInfoRaw, portRaw := "", ""
	if val.UserInfo != "" {
		userInfoRaw = val.UserInfo + "@"
	}
	if val.HasPort {
		portRaw = fmt.Sprintf(":%v", val.Port)
	}
	return userInfoRaw + val.Host + portRaw
}

// BoolValue
type BoolValue struct {
	Value     bool
	IsDefault bool
	Error     error
}

func NewBoolValueDefault(bool_default bool) *BoolValue {
	return &BoolValue{
		Value:     bool_default,
		IsDefault: true,
		Error:     nil,
	}
}

func NewBoolValue() *BoolValue {
	return NewBoolValueDefault(false)
}

func (val *BoolValue) Set(raw string) error {
	switch lower := strings.ToLower(raw); lower {
	case "true":
		val.Value = true
	case "false":
		val.Value = false
	default:
		return fmt.Errorf("Invalid value: %v", raw)
	}

	val.IsDefault = false
	return nil
}

func (val *BoolValue) String() string {
	switch val.Value {
	case true:
		return "true"
	case false:
		return "false"
	}
	return "Unknown"
}

// NetEndpointSetValue
type NetEndpointSetValue struct {
	Endpoints    map[string]*NetEndpointValue
	ValidSchemes []string
	IsDefault    bool
}

func NewNetEndpointSetValueDefault(validSchemes []string, netEndpointSet string) (*NetEndpointSetValue, error) {
	instance := &NetEndpointSetValue{
		Endpoints:    make(map[string]*NetEndpointValue, 0),
		ValidSchemes: validSchemes,
		IsDefault:    true,
	}

	if err := instance.Set(netEndpointSet); err != nil {
		return nil, err
	}
	instance.IsDefault = true

	return instance, nil
}

func NewNetEndpointSetValue(validSchemes []string) (*NetEndpointSetValue, error) {
	return NewNetEndpointSetValueDefault(validSchemes, "")
}

func (val *NetEndpointSetValue) Set(raw string) error {
	defer func() {
		val.IsDefault = false
	}()

	if raw == "" {
		return nil
	}
	for _, endpoint := range strings.Split(raw, ",") {
		parts := strings.SplitN(endpoint, "=", 2)
		if len(parts) < 2 {
			return fmt.Errorf("Invalid endpoint format: Endpoint name should be specified: %v", endpoint)
		}
		endpointValue, err := NewNetEndpointValueDefault(val.ValidSchemes, parts[1])
		if err != nil {
			return fmt.Errorf("Invalid endpoint format: %v", err.Error())
		}
		val.Endpoints[parts[0]] = endpointValue
	}
	return nil
}

func (val *NetEndpointSetValue) String() string {
	rawEndpointList := make([]string, 0, len(val.Endpoints))
	for k, v := range val.Endpoints {
		rawEndpointList = append(rawEndpointList, fmt.Sprintf("%v=%v", k, v))
	}
	return strings.Join(rawEndpointList, ",")
}
