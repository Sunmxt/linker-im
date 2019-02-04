package dig

type Registry interface {
	Service(name string) (Service, error)
	Node(name string) (*Node, error)
	Poll(func(*Notification)) (bool, error)
	Close()
	Publish(*Node) error
}
