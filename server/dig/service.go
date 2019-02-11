package dig

type Service interface {
	Name() string
	Nodes() []string
	Watch() error
	Publish(node *Node) error
}
