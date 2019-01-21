package svc

type BlobMapPersistPrimitive interface {
    Loads(tag string) (map[string][]byte, int64, error)
    Sets(tag string, kv map[string][]byte, version int64) error
    SetDefaults(tag string, kv map[string][]byte, version int64) error
    Dels(tag string, keys []string, version int64) error
}
