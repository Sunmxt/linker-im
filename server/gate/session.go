package gate

func (g *Gate) sessionKey(session string) string {
	var key string
	raw, ok := g.KeySession.Load(session)
	if !ok {
		return ""
	}
	if key, ok = raw.(string); !ok {
		return ""
	}
	return key
}
