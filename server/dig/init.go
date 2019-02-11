package dig

func init() {
	RegisterDriver("redis", &RedisConnector{})
}
