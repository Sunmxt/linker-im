package gate

func WrapErrorMessage(msg, id string) string {
	if !gate.config.DebugMode.Value {
		// Mask error message.
		msg = "Server raise an exception with ID \"" + id + "\""
	} else {
		// Add Request ID to error message
		msg += "[ID = " + id + "]"
	}
	return msg
}
