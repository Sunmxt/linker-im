package discover

var Drivers map[string]Connector = map[string]Connector{}

func RegisterDriver(driver string, connector Connector) error {
    if connector == nil {
        return ErrInvalidConnector
    }

    c, ok := Drivers[driver]
    if ok && c != nil {
        return ErrDriverExist
    }
    Drivers[driver] = connector

    return nil
}
