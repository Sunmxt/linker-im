package svc

import (
	"errors"
	"fmt"
	ilog "github.com/Sunmxt/linker-im/log"
	"strings"
)

var ErrNamespaceExist = errors.New("Namespace already exists.")
var ErrNamespaceMissing = errors.New("Namespace missing.")

func VaildNamespaceName(name string) bool {
	for _, runeValue := range name {
		if (runeValue >= '0' && runeValue <= '9') || (runeValue >= 'A' && runeValue <= 'Z') || (runeValue >= 'a' && runeValue <= 'z') || runeValue == '-' || runeValue == '_' {
			continue
		}
		return false
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

func (ns *SessionNamespace) logTraceNamespace() {
	ns.log.TraceLazy(func() string {
		currentNamespaces, version, err := ns.ns.List()
		if err != nil || currentNamespaces == nil {
			return "Failed to list session namespaces: " + err.Error()
		}
		return "Trace current session namespaces: " + strings.Join(currentNamespaces, ", ") + fmt.Sprintf("(version = %v)", version)
	})
}

func (ns *SessionNamespace) Append(namespaces []string) error {
	for _, name := range namespaces {
		if !VaildNamespaceName(name) {
			return fmt.Errorf("Invalid namespace name \"%v\"", name)
		}
	}

	version, err := ns.ns.Append(namespaces)
	if err != nil {
		return err
	}

	ns.log.Infof0("Session namespace \"" + strings.Join(namespaces, "\",\"") + "\"added." + fmt.Sprintf("(version = %v)", version))
	ns.logTraceNamespace()

	return nil
}

func (ns *SessionNamespace) List() ([]string, error) {
	namespaces, _, err := ns.ns.List()
	if err != nil {
		return nil, err
	}
	return namespaces, nil
}

func (ns *SessionNamespace) Remove(namespaces []string) error {
	_, err := ns.ns.Remove(namespaces)
	if err != nil {
		return err
	}
	return nil
}
