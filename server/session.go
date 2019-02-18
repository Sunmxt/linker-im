package server

type SessionPool interface {
	Get(string) (map[string]string, error)
	Register(map[string]string) (string, error)
	Remove(string) error
}
