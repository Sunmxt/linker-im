package discover

type Registry interface {
    Service(name string) (Service, error)
    Node(name string) (*Node, error)
    Poll() (bool, error)
    Close()
}


