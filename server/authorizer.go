package server

// Authorizer is responsible for granting/denying access to resources
// according to provided credentials.
type Authorizer interface {
	Connect(credentials []string, session map[string]string) (string, error)
	Auth(op interface{}, session map[string]string) error
}

// And Combinator
type AndAuthorizer struct {
	Authorizers []Authorizer
}

func AuthAnd(authorizers ...Authorizer) Authorizer {
	return &AndAuthorizer{
		Authorizers: append(make([]Authorizer, 0, len(authorizers)), authorizers...),
	}
}

func (comb *AndAuthorizer) Auth(op interface{}, session map[string]string) error {
	for _, authorizer := range comb.Authorizers {
		if err := authorizer.Auth(op, session); err != nil {
			return err
		}
	}
	return nil
}

func authorizerConnect(authorizers []Authorizer, credentials []string, session map[string]string) (string, error) {
	var key string
	var err error
	for _, authorizer := range authorizers {
		if key, err = authorizer.Connect(credentials, session); err != nil {
			return "", err
		}
	}
	return key, nil
}

func (comb *AndAuthorizer) Connect(credentials []string, session map[string]string) (string, error) {
	return authorizerConnect(comb.Authorizers, credentials, session)
}

// Or Combinator
type OrAuthorizer struct {
	Authorizers []Authorizer
}

func OrAuth(authorizers ...Authorizer) Authorizer {
	return &OrAuthorizer{
		Authorizers: append(make([]Authorizer, 0, len(authorizers)), authorizers...),
	}
}

func (comb *OrAuthorizer) Connect(credentials []string, session map[string]string) (string, error) {
	return authorizerConnect(comb.Authorizers, credentials, session)
}

func (comb *OrAuthorizer) Auth(op interface{}, session map[string]string) error {
	var err error
	for _, authorizer := range comb.Authorizers {
		if err = authorizer.Auth(op, session); err == nil {
			return nil
		}
	}
	return err
}
