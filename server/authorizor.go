package server

// Authorizor is responsible for granting/denying access to resources
// according to provided credentials.
type Authorizor interface {
	Auth(credentials map[string]string, args ...interface{}) error // Auth with credentials
	Identity() string                                              // Return authorizor identifier.
}

// And Combinator
type AndAuthorizor struct {
	Authorizors []Authorizor
	Name        string
}

func NamedAuthAnd(identifier string, authorizors ...Authorizor) Authorizor {
	return &AndAuthorizor{
		Name:        identifier,
		Authorizors: append(make([]Authorizor, 0, len(authorizors)), authorizors...),
	}
}

func AuthAnd(authorizors ...Authorizor) Authorizor {
	return NamedAuthAnd("And", authorizors...)
}

func (comb *AndAuthorizor) Auth(credentials map[string]string, args ...interface{}) error {
	var err error
	for _, authorizor := range comb.Authorizors {
		err = authorizor.Auth(credentials, args...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (comb *AndAuthorizor) Identity() string {
	return comb.Name
}

// Or Combinator
type OrAuthorizor struct {
	AuthorizorA Authorizor
	AuthorizorB Authorizor
	Name        string
}

func NamedOrAuth(identifier string, authorizorA, authorizorB Authorizor) Authorizor {
	return &OrAuthorizor{
		AuthorizorA: authorizorA,
		AuthorizorB: authorizorB,
		Name:        identifier,
	}
}

func OrAuth(authorizorA, authorizorB Authorizor) Authorizor {
	return NamedOrAuth("Or", authorizorA, authorizorB)
}

func (comb *OrAuthorizor) Auth(credentials map[string]string, args ...interface{}) error {
	if err := comb.AuthorizorA.Auth(credentials, args...); err == nil {
		return nil
	}
	return comb.AuthorizorB.Auth(credentials, args...)
}

func (comb *OrAuthorizor) Identity() string {
	return comb.Name
}
