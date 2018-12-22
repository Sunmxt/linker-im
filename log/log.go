package log

import (
    "github.com/sirupsen/logrus"
)

func Info(message string) {
    logrus.WithFields(logrus.Fields{
        "version": "v0.1.0",
    }).Info(message)
}

