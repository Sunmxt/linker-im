package svc

type DefaultAuthorizer struct{}

func (a *DefaultAuthorizer) Auth(namespace string, op uint16, session map[string]string) error {
	return nil
}

func (a *DefaultAuthorizer) Connect(namespace, credential string, session map[string]string) error {
	session["u"] = credential
	session["ns"] = namespace
	return nil
}

func (a *DefaultAuthorizer) Identifier(namespace string, session map[string]string) (string, error) {
	ident, _ := session["u"]
	return ident, nil
}
