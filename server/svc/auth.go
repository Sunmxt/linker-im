package svc

type DefaultAuthorizer struct{}

func (a *DefaultAuthorizer) Auth(op interface{}, session map[string]string) error { return nil }

func (a *DefaultAuthorizer) Connect(credentials []string, session map[string]string) (string, error) {
	if credentials == nil || len(credentials) < 1 {
		return "", nil
	}
	session["u"] = credentials[0]
	return credentials[0], nil
}
