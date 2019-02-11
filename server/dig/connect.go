package dig

func Connect(driver string, args ...interface{}) (Registry, error) {
	connector, ok := Drivers[driver]
	if !ok || connector == nil {
		return nil, ErrInvalidConnector
	}
	return connector.Connect(args...)
}

type Connector interface {
	Connect(...interface{}) (Registry, error)
}
