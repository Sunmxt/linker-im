package gate

import (
	"github.com/Sunmxt/linker-im/proto"
)

func (g *Gate) sessionKey(namespace, session string) string {
	key := namespace + "." + session
	raw, ok := g.KeySession.Load(key)
	if !ok {
		// Push key not found. Try connecting.
		if _, err := g.connect(&proto.ConnectV1{
			Credential: session,
			Type:       proto.CONN_SESSION,
			Namespace:  namespace,
		}); err != nil {
			return ""
		}
		raw, ok = g.KeySession.Load(key)
		if !ok {
			return ""
		}
	}
	if key, ok = raw.(string); !ok {
		return ""
	}
	return key
}
