package dig

type Registry interface {
	Service(name string) (Service, error)
	Node(name string) (*Node, error)
	Poll() (bool, error)
	Close()
	Publish(*Node) error
}
