package proto

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	ErrNilBuf         = errors.New("Nil buffer.")
	ErrBufferTooShort = errors.New("Buffer is too short.")
)

const (
	OP_DUMMY     = uint16(0)
	OP_RESPONSE  = uint16(1)
	OP_CONNECT   = uint16(2)
	OP_SUB       = uint16(3)
	OP_KEEPALIVE = uint16(4)
	OP_PUSH      = uint16(5)
	OP_MESSAGE   = uint16(6)
	OP_ERROR     = uint16(7)
	//OP_PULL        = uint16(6)
	//OP_GROUP_ENUM  = uint16(7)
	//OP_USER_ENUM   = uint16(8)
	//OP_NS_ENUM     = uint16(9)
	//OP_GROUP_ALTER = uint16(10)
	//OP_USER_ALTER  = uint16(11)
	//OP_NS_ALTER    = uint16(12)
)

type ProtocolUnit interface {
	Unmarshal([]byte) (uint, error)
	Marshal([]byte) error
	Len() int
	OpType() uint16
}

// Request
type Request struct {
	OpCode    uint16
	RequestID uint32
	Body      ProtocolUnit
}

func (r *Request) Marshal(buf []byte) error {
	if buf == nil {
		return ErrNilBuf
	}
	if r.Len() > len(buf) {
		return ErrBufferTooShort
	}
	r.OpCode = r.OpType()
	binary.BigEndian.PutUint16(buf[0:2], r.OpCode)
	binary.BigEndian.PutUint32(buf[2:6], r.RequestID)
	return r.Body.Marshal(buf[6:])
}

func (r *Request) OpType() uint16 {
	return r.Body.OpType()
}

func (r *Request) Len() int {
	return 6 + r.Body.Len()
}

func (r *Request) Unmarshal(raw []byte) (uint, error) {
	if len(raw) < 6 {
		return 0, ErrBufferTooShort
	}
	r.OpCode = r.Body.OpType()
	code := binary.BigEndian.Uint16(raw)
	if r.OpCode != code {
		return 0, fmt.Errorf("Invalid protocol type %v", code)
	}
	r.RequestID = binary.BigEndian.Uint32(raw[2:])
	consume, err := r.Body.Unmarshal(raw[6:])
	return 6 + consume, err
}

/////////////////////////////////////////////////////////
// Error :
/////////////////////////////////////////////////////////
type ErrorResponse struct {
	Err string
}

func (r *ErrorResponse) Marshal(buf []byte) error {
	binErr := []byte(r.Err)
	buf[0] = byte(len(binErr))
	copy(buf[1:], binErr)
	return nil
}

func (r *ErrorResponse) Unmarshal(raw []byte) error {
	return nil
}

func (r *ErrorResponse) Len() int {
	return 1 + len([]byte(r.Err))
}

func (r *ErrorResponse) OpType() uint16 {
	return OP_ERROR
}

/////////////////////////////////////////////////////////
// Connect :
// +-------+--------+---------+----...----+----...-----+
// |  Type | LEN_NS | LEN_CRE | Namespace | Credential |
// | (0-7) | (8-23) | (24-39) |   (40-N)  |   (N-M)    |
// +-------+--------+---------+----...----+----...-----+
/////////////////////////////////////////////////////////
func (r *ConnectV1) Marshal(buf []byte) error {
	binNS, binCre := []byte(r.Namespace), []byte(r.Credential)
	buf[0] = byte(r.Type)
	binary.BigEndian.PutUint16(buf[1:3], uint16(len(binNS)))
	binary.BigEndian.PutUint16(buf[3:5], uint16(len(binCre)))
	copy(buf[5:], binNS)
	copy(buf[5+len(binNS):], binCre)
	return nil
}

func (r *ConnectV1) Len() int {
	return 5 + len([]byte(r.Namespace)) + len([]byte(r.Credential))
}

func (r *ConnectV1) OpType() uint16 {
	return OP_CONNECT
}

func (r *ConnectV1) Unmarshal(buf []byte) (uint, error) {
	if len(buf) < 5 {
		return 0, ErrBufferTooShort
	}
	r.Type = uint8(buf[0])
	lenNS, lenCre := binary.BigEndian.Uint16(buf[1:]), binary.BigEndian.Uint16(buf[3:])
	if uint16(len(buf)) < 5+lenNS+lenCre {
		return 5, ErrBufferTooShort
	}
	r.Namespace = string(buf[3 : 3+lenNS])
	r.Credential = string(buf[3+lenNS : 3+lenNS+lenCre])
	return uint(3 + lenNS + lenCre), nil
}

/////////////////////////////////////////////////////////
// ConnectResultV1 :
/////////////////////////////////////////////////////////
func (r *ConnectResultV1) Unmarshal(raw []byte) (uint, error) {
	return 0, nil // Not implemented.
}

func (r *ConnectResultV1) Marshal(buf []byte) error {
	binAuthErr, binSession := []byte(r.AuthError), []byte(r.Session)
	buf[0] = byte(len(binAuthErr))
	binary.BigEndian.PutUint16(buf[1:], uint16(len(binSession)))
	copy(buf[3:], binAuthErr)
	copy(buf[3+len(binAuthErr):], binSession)
	return nil
}

func (r *ConnectResultV1) Len() int {
	return 3 + len([]byte(r.AuthError)) + len([]byte(r.Session))
}

func (er *ConnectResultV1) OpType() uint16 {
	return OP_RESPONSE
}

/////////////////////////////////////////////////////////
// Sub/Unsub :
/////////////////////////////////////////////////////////
func (r *Subscription) Marshal(buf []byte) error {
	ns, session, grp := []byte(r.Namespace), []byte(r.Session), []byte(r.Group)
	buf[0] = byte(r.Op)
	binary.BigEndian.PutUint16(buf[1:3], uint16(len(ns)))
	binary.BigEndian.PutUint16(buf[3:5], uint16(len(session)))
	binary.BigEndian.PutUint16(buf[5:7], uint16(len(grp)))
	copy(buf[7:], ns)
	copy(buf[7+len(ns):], session)
	copy(buf[7+len(ns)+len(session):], grp)
	return nil
}

func (r *Subscription) Len() int {
	return 7 + len([]byte(r.Namespace)) + len([]byte(r.Session)) + len([]byte(r.Group))
}

func (r *Subscription) OpType() uint16 {
	return OP_SUB
}

func (r *Subscription) Unmarshal(raw []byte) (uint, error) {
	if len(raw) < 7 {
		return 0, ErrBufferTooShort
	}
	r.Op = byte(raw[0])
	lenNS, lenSession, lenGrp := binary.BigEndian.Uint16(raw[1:]), binary.BigEndian.Uint16(raw[3:]), binary.BigEndian.Uint16(raw[5:])
	if uint16(len(raw)) < 7+lenNS+lenSession {
		return 7, ErrBufferTooShort
	}
	cnt := uint16(7)
	r.Namespace = string(raw[cnt : cnt+lenNS])
	cnt += lenNS
	r.Session = string(raw[cnt : cnt+lenSession])
	cnt += lenSession
	r.Group = string(raw[cnt : cnt+lenGrp])
	cnt += lenGrp
	return uint(cnt), nil
}

/////////////////////////////////////////////////////////
// MessageBody :
/////////////////////////////////////////////////////////
func (r *MessageBody) Marshal(buf []byte) error {
	binRaw, binGrp := []byte(r.Raw), []byte(r.Group)
	binary.BigEndian.PutUint16(buf, uint16(len(r.Group)))
	binary.BigEndian.PutUint32(buf[2:], uint32(len(r.Raw)))
	copy(buf[6:], binGrp)
	copy(buf[6+len(binGrp):], binRaw)
	return nil
}

func (r *MessageBody) Unmarshal(raw []byte) (uint, error) {
	if len(raw) < 6 {
		return 0, ErrBufferTooShort
	}
	lenGrp, lenRaw := binary.BigEndian.Uint16(raw), binary.BigEndian.Uint32(raw[2:])
	if uint(len(raw)) < 6+uint(lenGrp)+uint(lenRaw) {
		return 6, ErrBufferTooShort
	}
	r.Group = string(raw[6 : 6+lenGrp])
	r.Raw = string(raw[6+lenGrp : 6+uint(lenGrp)+uint(lenRaw)])
	return 6 + uint(lenGrp) + uint(lenRaw), nil
}

func (r *MessageBody) Len() int {
	return 6 + len([]byte(r.Group)) + len([]byte(r.Raw))
}

func (r *MessageBody) OpType() uint16 {
	return OP_PUSH
}

/////////////////////////////////////////////////////////
// Push :
/////////////////////////////////////////////////////////
func (r *MessagePushV1) Marshal(buf []byte) error {
	return nil
}

func (r *MessagePushV1) Unmarshal(raw []byte) (uint, error) {
	return 0, nil
}

func (r *MessagePushV1) Len() int {
	return 0
}

func (r *MessagePushV1) OpType() uint16 {
	return OP_PUSH
}
