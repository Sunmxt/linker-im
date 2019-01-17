package svc

type VCCSPersistCapabilities struct {
	Append bool
	Remove bool
	List   bool
	Update bool
}

type VCCSPersistPrimitive interface {
	Capabilities() *VCCSPersistCapabilities

	List() ([]string, int64, error)
	Append([]string, int64) (bool, error)
	Remove([]string, int64) (bool, error)
	Update([]string, int64) (bool, error)
}
