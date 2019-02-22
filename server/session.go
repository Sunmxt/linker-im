package server

type SessionPool interface {
	Get(string, string) (map[string]string, error)
	Register(string, map[string]string) (string, error)
	Remove(string, string) error
}
