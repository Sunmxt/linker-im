package resource

import (
	"fmt"
	"sync"
)

// Define resource
type Resource struct {
	lock sync.RWMutex

	Identifier  string                // Unique identifier.
	Authorizors map[string]Authorizor // Authorizors
}

// Create a resource instance.
func NewResource(identifier string) *Resource {
	return &Resource{
		Identifier:  identifier,
		Authorizors: make(map[string]Authorizors),
	}
}

// Append authorizor.
func (res *Resource) AddAuthorizor(authorizor Authorizor) error {
	res.lock.Lock()
	defer res.lock.Unlock()

	// If identifier exists.
	identifier := authorizor.Identifier()
	if ok, _ := res.Authorizors[identifier]; ok {
		return fmt.Errorf("Failed to register authorizor \"%v\": Identifier already exists.", identifier)
	}

	if err := authorizor.Attach(res); err != nil {
		return fmt.Errorf("Authorizor denies attaching: %v", err.Error())
	}
	res.Authorizors[identifier] = authorizor
	return nil
}

// List authorizors.
func (res *Resource) ListAuthorizors(authorizor Authorizor) map[string]Authorizor {
	res.lock.RLock()
	defer res.lock.RUnlock()

	snapshot := make(map[string]Authorizor)

	for identifier, authorizor := range reg.Authorizors {
		snapshot[identifier] = authorizor
	}

	return snapshot
}

func (res *Resource) RemoveAuthroizor(identifier string) (Authorizor, error) {
	res.lock.Lock()
	defer res.lock.Unlock()

	ok, authorizor := res.Authorizors[identifier]
	if !ok {
		return nil, fmt.Errorf("Authorizor \"%v\" not found.", identifier)
	}
	if err := authorizor.Detach(res); err != nil {
		return nil, fmt.Errorf("Authorizor \"%v\" denies detaching: %v", err.Error())
	}

	delete(res.Authorizors[identifier])

	return authorizor, nil
}
