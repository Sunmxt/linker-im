package svc

import (
	"errors"
	ilog "github.com/Sunmxt/linker-im/log"
	"github.com/gomodule/redigo/redis"
	"sync"
	"time"
)

var ErrNamespaceMissing = errors.New("Namespace missing.")
var ErrInvalidNamespaceName = errors.New("Invalid namespace name.")

func VaildNamespaceName(name string) bool {
	for _, runeValue := range name {
		if (runeValue >= '0' && runeValue <= '9') || (runeValue >= 'A' && runeValue <= 'Z') || (runeValue >= 'a' && runeValue <= 'z') || runeValue == '-' || runeValue == '_' {
			continue
		}
		return false
	}
	return true
}

type Model struct {
	blobMapPool sync.Map
	Pool        *redis.Pool
	Log         *ilog.Logger
	Prefix      string
}

type ModelBlobMapReference struct {
	Map        *BlobMap
	LastAccess time.Time
}

func NewModel(pool *redis.Pool, prefix string) *Model {
	model := &Model{
		Pool:   pool,
		Log:    ilog.NewLogger(),
		Prefix: prefix,
	}
	model.Log.Fields["entity"] = "model"
	return model
}

// Optimize clean idle blobmap from pool.
func (m *Model) Optimize(before time.Time) {
	m.blobMapPool.Range(func(key, value interface{}) bool {
		ref, ok := value.(*ModelBlobMapReference)
		if !ok || ref.LastAccess.Before(before) {
			m.blobMapPool.Delete(key)
		}
		return true
	})
}

func (m *Model) Subscribe(namespace, group string, users []string) error {
	bm := m.GetBlobMap("group."+namespace+"."+group, 0, nil)
	binParam := NewSubscriptionMetadata().Serialize()
	kv := make(map[string][]byte, len(users))
	for _, user := range users {
		kv[user] = binParam
	}
	if _, err := bm.SetDefaults(kv); err != nil {
		return err
	}
	// Update mapping.
	return nil
}

func (m *Model) setMetadata(key string, metas map[string][]byte, isDefault bool) error {
	var version int64
	var err error

	bm := m.GetBlobMap(key, 0, nil)
	if isDefault {
		version, err = bm.SetDefaults(metas)
	} else {
		version, err = bm.Sets(metas)
	}
	if err != nil {
		m.Log.Error("Failed to set default metadata for " + "\"" + key + "\"" + err.Error())
		return err
	}
	m.Log.Infof1("Set default metadata for key \"%v\". (version = %v)", key, version)

	return nil
}

func (m *Model) delMetadata(key string, metas []string) error {
	bm := m.GetBlobMap(key, 0, nil)
	version, err := bm.Dels(metas)

	if err != nil {
		m.Log.Error("Failed to delete metadata for " + "\"" + key + "\": " + err.Error())
		return err
	}
	m.Log.Infof1("Delete metadata for key \"%v\". (version = %v)", key, version)

	return nil
}

func (m *Model) listMetadata(key string) ([]string, error) {
	bm := m.GetBlobMap(key, 0, nil)
	keys, version, err := bm.Keys()
	if err != nil {
		m.Log.Error("Failed to list metadata for " + "\"" + key + "\": " + err.Error())
		return nil, err
	}
	m.Log.Infof1("List metadata for key \"%v\". (version = %v)", key, version)

	return keys, nil
}

func (m *Model) getMetadata(key string, mapKeys []string) ([][]byte, error) {
	bm := m.GetBlobMap(key, 0, nil)
	binarys, version, err := bm.Gets(mapKeys)
	if err != nil {
		m.Log.Error("Failed to get metadata for " + "\"" + key + "\": " + err.Error())
		return nil, err
	}
	m.Log.Infof1("Get metadata for key \"%v\". (version = %v)", key, version)
	return binarys, nil
}

func (m *Model) GetNamespaceMetadata(namespaces []string) ([]*NamespaceMetadata, error) {
	bins, err := m.getMetadata("namespaces", namespaces)
    if err != nil {
		return nil, err
	}
	metas := make([]*NamespaceMetadata, len(namespaces), len(namespaces))
	for idx, key := range namespaces {
		if bin := bins[idx]; bin != nil {
			meta := &NamespaceMetadata{}
			if err = meta.Unserialize(bin); err != nil {
				m.Log.Fatal("Broken metadata of namespace \"" + key + "\":" + err.Error())
			} else {
				metas[idx] = meta
			}
		}
	}
	return metas, nil
}

func (m *Model) GetGroupMetadata(namespace string, groups []string) ([]*GroupMetadata, error) {
	bins, err := m.getMetadata("groups."+namespace, groups)
    if err != nil {
		return nil, err
	}
	metas := make([]*GroupMetadata, len(groups), len(groups))
	for idx, key := range groups {
		if bin := bins[idx]; bin != nil {
			meta := &GroupMetadata{}
			if err = meta.Unserialize(bin); err != nil {
				m.Log.Fatal("Broken metadata of group \"" + key + "\" in namespace \"" + namespace + "\"" + ":" + err.Error())
			} else {
				metas[idx] = meta
			}
		}
	}
	return metas, nil
}

func (m *Model) GetUserMetadata(namespace string, users []string) ([]*UserMetadata, error) {
	bins, err := m.getMetadata("users."+namespace, users)
    if err != nil {
		return nil, err
	}
	metas := make([]*UserMetadata, len(users), len(users))
	for idx, key := range users {
		if bin := bins[idx]; bin != nil {
			meta := &UserMetadata{}
			if err = meta.Unserialize(bin); err != nil {
				m.Log.Fatal("Broken metadata of user \"" + key + "\" in namespace \"" + namespace + "\"" + ":" + err.Error())
			} else {
				metas[idx] = meta
			}
		}
	}
	return metas, nil
}

func (m *Model) SetNamespaceMetadata(metas map[string]*NamespaceMetadata, isDefault bool) error {
	bins := make(map[string][]byte, len(metas))
	for k, v := range metas {
		bins[k] = v.Serialize()
	}
	return m.setMetadata("namespaces", bins, isDefault)
}

func (m *Model) SetGroupMetadata(namespaces string, metas map[string]*GroupMetadata, isDefault bool) error {
	bins := make(map[string][]byte, len(metas))
	for k, v := range metas {
		bins[k] = v.Serialize()
	}
	return m.setMetadata("groups."+namespaces, bins, isDefault)
}

func (m *Model) SetUserMetadata(namespaces string, metas map[string]*UserMetadata, isDefault bool) error {
	bins := make(map[string][]byte, len(metas))
	for k, v := range metas {
		bins[k] = v.Serialize()
	}
	return m.setMetadata("users."+namespaces, bins, isDefault)
}

func (m *Model) DeleteNamespaceMetadata(namespaces []string) error {
	return m.delMetadata("namespaces", namespaces)
}

func (m *Model) DeleteGroupMetadata(namespace string, groups []string) error {
	return m.delMetadata("groups."+namespace, groups)
}

func (m *Model) DeleteUserMetadata(namespace string, users []string) error {
	return m.delMetadata("users."+namespace, users)
}

func (m *Model) ListNamespace() ([]string, error) {
	return m.listMetadata("namespaces")
}

func (m *Model) ListUser(namespace string) ([]string, error) {
	return m.listMetadata("users." + namespace)
}

func (m *Model) ListGroup(namespace string) ([]string, error) {
	return m.listMetadata("groups." + namespace)
}

func (m *Model) GetBlobMap(key string, timeout int, primitive BlobMapPersistPrimitive) *BlobMap {
	var ref *ModelBlobMapReference

	newBlobMap := func() *ModelBlobMapReference {
		if ref == nil {
			ref = &ModelBlobMapReference{
				Map: NewBlobMap(m.Pool, m.Prefix, key, timeout, primitive),
			}
			m.Log.Infof1("Add new blobmap \"" + key + "\" to pool.")
		}
		return ref
	}

	for ref == nil {
		raw, loaded := m.blobMapPool.Load(key)

		if !loaded {
			raw, loaded = m.blobMapPool.LoadOrStore(key, newBlobMap())
			if !loaded { // Stored by me.
				break
			}
		}

		ref, loaded = raw.(*ModelBlobMapReference)
		if !loaded { // Wrong type. Force to replace.
			m.blobMapPool.Store(key, newBlobMap())
		}
		break
	}

	// update access time.
	ref.LastAccess = time.Now()

	return ref.Map
}
