package discover

import (
    "errors"
)

var ErrDriverExist = errors.New("Driver exists.")
var ErrDriverMissing = errors.New("Driver missing.")
var ErrInvalidConnector = errors.New("Invalid connector.")
var ErrInvalidArguments = errors.New("Invalid arguments.")
var ErrServiceNotFound = errors.New("Service not found.")
