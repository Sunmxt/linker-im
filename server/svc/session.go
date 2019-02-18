package svc

import (
	"strings"
)

type DefaultSessionPool struct{}

func (s *DefaultSessionPool) Get(key string) (map[string]string, error) {
	parts, session := strings.SplitN(key, ".", 2), map[string]string{}
	if len(parts) != 2 {
		return session, nil
	}
	session["ns"] = parts[0]
	session["u"] = parts[1]
	return session, nil
}

func (s *DefaultSessionPool) Register(session map[string]string) (string, error) {
	key, _ := session["u"]
	ns, _ := session["ns"]
	return ns + "." + key, nil
}

func (s *DefaultSessionPool) Remove(key string) error { return nil }
