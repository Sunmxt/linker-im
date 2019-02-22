package server

// Authorizer is responsible for granting/denying access to resources
// according to provided credentials.
type Authorizer interface {
	Connect(namespace, credential string, session map[string]string) error
	Auth(namespace string, op uint16, session map[string]string) error
	Identifier(namespace string, session map[string]string) (string, error)
}

// And Combinator
//type AndAuthorizer struct {
//	Authorizers []Authorizer
//}
//
//func AuthAnd(authorizers ...Authorizer) Authorizer {
//	return &AndAuthorizer{
//		Authorizers: append(make([]Authorizer, 0, len(authorizers)), authorizers...),
//	}
//}
//
//func (comb *AndAuthorizer) Auth(namespace string, op uint16, session map[string]string) error {
//	for _, authorizer := range comb.Authorizers {
//		if err := authorizer.Auth(op, namespace, session); err != nil {
//			return err
//		}
//	}
//	return nil
//}
//
//func authorizerConnect(authorizers []Authorizer, namespace, credential string, session map[string]string) error {
//	for _, authorizer := range authorizers {
//		if err := authorizer.Connect(namespace, credential, session); err != nil {
//			return err
//		}
//	}
//	return nil
//}
//
//func (comb *AndAuthorizer) Connect(namespace, credential string, session map[string]string) error {
//	return authorizerConnect(comb.Authorizers, namespace, credential, session)
//}
//
//// Or Combinator
//type OrAuthorizer struct {
//	Authorizers []Authorizer
//}
//
//func OrAuth(authorizers ...Authorizer) Authorizer {
//	return &OrAuthorizer{
//		Authorizers: append(make([]Authorizer, 0, len(authorizers)), authorizers...),
//	}
//}
//
//func (comb *OrAuthorizer) Connect(namespace, credential string, session map[string]string) error {
//	return authorizerConnect(comb.Authorizers, namespace, credential, session)
//}
//
//func (comb *OrAuthorizer) Auth(namespace string, op uint16, session map[string]string) error {
//	var err error
//	for _, authorizer := range comb.Authorizers {
//		if err = authorizer.Auth(namespace, op, session); err == nil {
//			return nil
//		}
//	}
//	return err
//}
