package svc

type DefaultSessionPool struct{}

func (s *DefaultSessionPool) Get(namespace, key string) (map[string]string, error) {
	return map[string]string{"u": key, "ns": namespace}, nil
}

func (s *DefaultSessionPool) Register(namespace string, session map[string]string) (string, error) {
	key, _ := session["u"]
	return key, nil
}

func (s *DefaultSessionPool) Remove(namespace, key string) error { return nil }
