package svc

import (
	"encoding/binary"
	"errors"
)

//
var ErrLengthUnmatched = errors.New("Wrong length of raw bytes.")

type SerialiableEntity interface {
	Serialize() []byte
	Unserialize(stream []byte) error
}

type NamespaceMetadata struct {
}

func (dat *NamespaceMetadata) Serialize() []byte {
	return make([]byte, 0)
}

func (dat *NamespaceMetadata) Unserialize([]byte) error {
	return nil
}

func NewDefaultNamespaceMetadata() *NamespaceMetadata {
	return &NamespaceMetadata{}
}

type GroupMetadata struct {
}

func (dat *GroupMetadata) Serialize() []byte {
	return make([]byte, 0)
}

func (dat *GroupMetadata) Unserialize([]byte) error {
	return nil
}

func NewDefaultGroupMetadata() *GroupMetadata {
	return &GroupMetadata{}
}

type UserMetadata struct {
}

func (dat *UserMetadata) Serialize() []byte {
	return make([]byte, 0)
}

func (dat *UserMetadata) Unserialize([]byte) error {
	return nil
}

func NewDefaultUserMetadata() *GroupMetadata {
	return &GroupMetadata{}
}

type SubscriptionMetadata struct {
	NotAfter int64
}

func NewSubscriptionMetadata() *SubscriptionMetadata {
	return &SubscriptionMetadata{}
}
func (dat *SubscriptionMetadata) Serialize() []byte {
	bin := make([]byte, 8)
	binary.LittleEndian.PutUint64(bin, uint64(dat.NotAfter))
	return bin
}

func (dat *SubscriptionMetadata) Unserialize(bin []byte) error {
	if len(bin) != 8 {
		return ErrLengthUnmatched
	}
	nft := binary.LittleEndian.Uint64(bin)
	dat.NotAfter = int64(nft)
	return nil
}
