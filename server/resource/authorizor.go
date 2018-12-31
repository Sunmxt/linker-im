package resource

// Authorizor is responsible for granting/denying access to resources
// according to provided credentials.
type Authorizor interface {
	Auth(resource *Resource, credentials map[string]string, args ...interface{}) error // Auth with credentials
	Attach(resource *Resource) error                                                   // Called when authorizor is attaching to resource.
	Detach(resource *Resource) error                                                   // Called when attaching is detathing from resource.
	Identity() string                                                                  // Return authorizor identifier.
}
