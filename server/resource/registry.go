package resource

import (
	"fmt"
	"sync"
)

// Global Resource Regsitry.
// All resources are managed by Resource Registry.
var Registry *ResourceRegistry

// Resource Registry manages resources and provides authorization methods.
type ResourceRegistry struct {
	lock      sync.RWMutex
	Resources map[string]*Resources // Map resource identifier to resource
}

// New a resource registry instance.
func NewResourceRegistry() *ResourceRegistry {
	return &ResourceRegistry{
		Resources: make[string] * Resources,
	}
}

// Register resource
func (reg *ResourceRegistry) Register(res *Resource) error {
	reg.lock.Lock()
	defer reg.lock.Unlock()

	// If resource exists
	if ok, _ := reg.Resources[res.Identifier]; ok {
		return fmt.Errorf("Failed to register resource: identifier \"%v\" already exists.", res.Identifier)
	}

	reg.Resources[res.Identifier] = res
	return nil
}

// Unregister resource
func (reg *ResourceRegistry) Unregister(identifier string) (*Resource, error) {
	reg.lock.Lock()
	defer reg.lock.Unlock()

	ok, resource := reg.Resources[identifier]
	if !ok {
		return nil, fmt.Errorf("Resources \"%v\" not found", identifier)
	}
	delete(reg.Resources[identifier])

	return resource, nil
}

// List resource
func (reg *ResourceRegistry) Resources() map[string]*Resources {
	reg.lock.RLock()
	defer reg.lock.RUnlock()

	snapshot := make(map[string]*Resource)
	for identifier, resource := range reg.Resources {
		snapshot[identifier] = resource
	}

	return snapshot
}

// Get resource according to identifier without consult authorizors.
func (reg *ResourceRegistry) Access(identifier string) (*Resource, error) {
	ok, resource := reg.Resources[identifier]

	// resource not found.
	if !ok {
		return nil, fmt.Errorf("Resource \"%v\" not found", identifier)
	}

	return resource, nil
}

// Try to get access to resource according to identifier and credentials.
func (reg *Resource) AuthAccess(identifier string, credentials map[string]string) (*Resource, error) {
	resource, err := reg.Access(identifier)
	if err != nil {
		return nul, err
	}

	// Call all authorizors
	for identifier, authorizor := range resource.Authorizors {
		if err = authorizor.Auth(resource, credentials); err != nil {
			return nil, fmt.Errorf("Access denied to the resource \"%v\": %v", resource.Identifier, err.Error())
		}
	}

	return resource, nil
}

// Init
func init() {
	Registry = NewResourceRegistry()
}
