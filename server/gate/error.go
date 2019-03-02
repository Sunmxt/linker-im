package gate

import (
	"github.com/Sunmxt/linker-im/log"
)

func WrapErrorMessage(msg, id string, ignoreDebug bool) string {
	if !gate.config.DebugMode.Value && !ignoreDebug {
		// Mask error message.
		msg = "Server raise an exception with ID \"" + id + "\""
	} else {
		// Add Request ID to error message
		msg += "[ID = " + id + "]"
	}
	log.Info2(msg)
	return msg
}
