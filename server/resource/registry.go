package resource

import (
	"fmt"
	"sync"
)

// Global Resource Registry.
// All resources are managed by Resource Registry.
var Registry *ResourceRegistry

// Resource Registry manages resources and provides authorization methods.
type ResourceRegistry struct {
	lock      sync.RWMutex
	Resources map[string]*Resource // Map resource identifier to resource
}

// New a resource registry instance.
func NewResourceRegistry() *ResourceRegistry {
	return &ResourceRegistry{
		Resources: make(map[string]*Resource),
	}
}

// Register resource
func (reg *ResourceRegistry) Register(identifier string, entity interface{}) error {
	reg.lock.Lock()
	defer reg.lock.Unlock()

	// If resource exists
	if _, ok := reg.Resources[identifier]; ok {
		return fmt.Errorf("Failed to register resource: resource \"%v\" already exists.", identifier)
	}

	reg.Resources[identifier] = &Resource{
		Identifier: identifier,
		Authorizor: nil,
		Entity:     entity,
	}
	return nil
}

// Unregister resource
func (reg *ResourceRegistry) Unregister(identifier string) (interface{}, error) {
	reg.lock.Lock()
	defer reg.lock.Unlock()

	resource, ok := reg.Resources[identifier]
	if !ok {
		return nil, fmt.Errorf("Resources \"%v\" not found", identifier)
	}
	delete(reg.Resources, identifier)

	return resource.Entity, nil
}

// List resource
func (reg *ResourceRegistry) ListResources() []string {
	reg.lock.RLock()
	defer reg.lock.RUnlock()

	snapshot := make([]string, 0, len(reg.Resources))
	for identifier, _ := range reg.Resources {
		snapshot = append(snapshot, identifier)
	}

	return snapshot
}

// Get resource according to identifier without consult authorizors.
func (reg *ResourceRegistry) Access(identifier string) (interface{}, error) {
	resource, err := reg.access(identifier)
	return resource.Entity, err
}

func (reg *ResourceRegistry) access(identifier string) (*Resource, error) {
	resource, ok := reg.Resources[identifier]
	// resource not found.
	if !ok {
		return nil, fmt.Errorf("Resource \"%v\" not found", identifier)
	}

	return resource, nil
}

// Try to get access to resource according to identifier and credentials.
func (reg *ResourceRegistry) AuthAccess(identifier string, credentials map[string]string, args ...interface{}) (interface{}, error) {
	resource, err := reg.access(identifier)
	if err != nil {
		return nil, err
	}

	// Call authorizor
	if err = resource.Authorizor.Auth(resource, credentials, args...); err != nil {
		return nil, err
	}

	return resource.Entity, nil
}

func init() {
	Registry = NewResourceRegistry()
}
