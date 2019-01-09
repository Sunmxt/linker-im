package svc

import (
	"errors"
	ilog "github.com/Sunmxt/linker-im/log"
)

var ErrNamespaceExist = errors.New("Namespace already exists.")
var ErrNamespaceMissing = errors.New("Namespace missing.")

func VaildNamespaceName(name string) bool {
	for _, runeValue := range name {
		if runeValue < 'A' || runeValue > 'z' || (runeValue < 'a' && runeValue > 'Z') || (runeValue != '-' && runeValue != '_') {
			return false
		}
	}
	return true
}

type SessionNamespace struct {
	ns  *VCCS
	log *ilog.Logger
}

func NewSessionNamespace(network, address, prefix string, timeout, maxWorker int, primitive VCCSPersistPrimitive) *SessionNamespace {
	instance := &SessionNamespace{
		ns:  NewVCCS(network, address, prefix, "session_namespace", timeout, maxWorker, primitive),
		log: ilog.NewLogger(),
	}
	log.Fields["entity"] = "session-namespace"
	return instance
}
