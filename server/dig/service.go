package dig

type Service interface {
	Nodes() []string
	Watch() error
	Publish(node *Node) error
}
