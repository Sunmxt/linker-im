package resource

// Resource
type Resource struct {
	Identifier string // Unique identifier.
	Authorizor
	Entity interface{}
}
