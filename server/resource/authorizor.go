package resource

// Authorizor is responsible for granting/denying access to resources
// according to provided credentials.
type Authorizor interface {
	Auth(resource *Resource, credentials map[string]string, args ...interface{}) error // Auth with credentials
	Attach(resource *Resource) error                                                   // Called when authorizor is attaching to resource.
	Detach(resource *Resource)                                                         // Called when attaching is detathing from resource.
	Identity() string                                                                  // Return authorizor identifier.
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

func (comb *AndAuthorizor) Auth(resource *Resource, credentials map[string]string, args ...interface{}) error {
	var err error
	for _, authorizor := range comb.Authorizors {
		err = authorizor.Auth(resource, credentials, args...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (comb *AndAuthorizor) Attach(resource *Resource) error {
	var err error = nil
	attached := make([]Authorizor, 0, len(comb.Authorizors))
	for _, authorizor := range comb.Authorizors {
		err = authorizor.Attach(resource)
		if err != nil {
			break
		}
		attached = append(attached, authorizor)
	}
	if err != nil {
		for _, authorizor := range attached {
			authorizor.Detach(resource)
		}
	}
	return err
}

func (comb *AndAuthorizor) Detach(resource *Resource) {
	for _, authorizor := range comb.Authorizors {
		authorizor.Detach(resource)
	}
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

func (comb *OrAuthorizor) Auth(resource *Resource, credentials map[string]string, args ...interface{}) error {
	if err := comb.AuthorizorA.Auth(resource, credentials, args...); err == nil {
		return nil
	}
	return comb.AuthorizorB.Auth(resource, credentials, args...)
}

func (comb *OrAuthorizor) Attach(resource *Resource) error {
	if err := comb.AuthorizorA.Attach(resource); err != nil {
		return err
	}
	if err := comb.AuthorizorB.Attach(resource); err != nil {
		comb.AuthorizorA.Detach(resource)
		return err
	}
	return nil
}

func (comb *OrAuthorizor) Detach(resource *Resource) {
	comb.AuthorizorA.Detach(resource)
	comb.AuthorizorB.Detach(resource)
}

func (comb *OrAuthorizor) Identity() string {
	return comb.Name
}
