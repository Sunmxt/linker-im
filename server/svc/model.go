package svc

import (
	"errors"
	ilog "github.com/Sunmxt/linker-im/log"
    guuid "github.com/satori/go.uuid"
    "github.com/Sunmxt/linker-im/proto"
    "sync"
    "time"
    "strings"
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

// Optimize clean idle blobmap from pool.
func (m *Model) Optimize(before time.Time) {
    m.Map.Range(func (key, value interface{}) bool {
        ref, ok := value.(*ModelBlobMapReference)
        if !ok || ref.LastAccess.Before(before) {
            m.Map.Delete(key)
        }
    })
}

type Model struct {
    blobMapPool  sync.Map
    Pool         *redis.Pool
    Log          *ilog.Logger
}

type ModelBlobMapReference struct {
    Map *BlobMap
    LastAccess time.Time
}

func (m *Model) Subscribe(namespace, group string, users []string) error {
    bm, err := m.GetBlobMap(b.prefix + "{group." + namespace + "." + group + "}")
    if err != nil {
        return err
    }
    binParam := string(NewDefaultSessionMetadata().Serialize())
    kv := make(map[string]string, len(user))
    for user := range users {
        kv[user] = binParam
    }
    if _, err = bm.SetDefaults(kv); err != nil {
        return err
    }
    // Update mapping.
    return nil
}

func (m *Model) setMetadata(key string, metas map[string]SerizliableEntity, isDefault bool) error {
    var version int64

    bm, err := m.GetBlobMap(key)
    if err != nil {
        return err
    }

    binData := make(map[string][]byte, len(metas))
    for k, v := range metas {
        binData[k] = v.Serialize()
    }

    if isDefault {
        version, err = bm.SetDefaults(binData)
    } else {
        version, err = bm.Sets(binData)
    }
    if err != nil {
        m.Log.Error("Failed to set default metadata for " + "\"" + key + "\"" + err.Error())
        return err
    }
    m.Log.Infof1("Set default metadata for key \"%v\". (version = %v)", key, version)

    return nil
}

func (m *Model) delMetadata(key string, metas []string) error {
    var version int64

    bm, err := m.GetBlobMap(key)
    if err != nil {
        return err
    }
    version, err = bm.Dels(metas)

    if err != nil {
        m.Log.Error("Failed to delete metadata for " + "\"" + key + "\"" + err.Error())
        return err
    }
    m.Log.Infof1("Delete metadata for key \"%v\". (version = %v)", key, version)

    return nil
}

func (m *Model) getMetadata(key string, metas map[string][]byte) error {
    var version int64

    bm, err := m.GetBlobMap(key)
    if err != nil {
        return err
    }

    keys := make([]string, 0, len(metas))
    for k, _ := range metas {
        keys = append(keys, k)
    }
    binarys, version, err := bm.Gets(keys)
    if err != nil {
        m.Log.Error("Failed to get metadata for " + "\"" + key + "\"" + err.Error())
        return err
    }
    m.Log.Infof1("Get metadata for key \"%v\". (version = %v)", key, version)
    for idx, key := range keys {
        bin := binarys[idx]
        if bin != nil {
            metas[key] = bin
        }
    }

    return nil
}

func (m *Model) GetNamespaceMetadata(namespaces []string) ([]*NamespaceMetadata, error) {
    var err error

    kv := make(map[string][]byte, len(namespaces))
    for key := range namespaces {
        kv[key] = nil
    }
    if err = m.getMetadata(s.prefix + "{namespaces}", kv); err != nil {
        return nil, err
    }
    metas := make([]*NamespaceMetadata, len(namespaces))
    for idx, key := range namespaces {
        bin, ok := kv[key]
        if ok && bin != nil {
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
    var err error

    kv := make(map[string][]byte, len(groups))
    for key := range groups {
        kv[key] = nil
    }
    if err = m.getMetadata(s.prefix + "{groups." + namespace + "}", kv); err != nil {
        return nil, err
    }
    metas := make([]*GroupMetadata, len(groups))
    for idx, key := range groups {
        bin, ok := kv[key]
        if ok && bin != nil {
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
    var err error

    kv := make(map[string][]byte, len(groups))
    for key := range users {
        kv[key] = nil
    }
    if err = m.getMetadata(s.prefix + "{users." + namespace + "}", kv); err != nil {
        return nil, err
    }
    metas := make([]*UserMetadata, len(users))
    for idx, key := range users {
        bin, ok := kv[key]
        if ok && bin != nil {
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

func (m *Model) SetNamespacesMetadata(namespaces map[string]*NamespaceMetadata, isDefault bool) error {
    return m.setMetadata(s.prefix + "{namespaces}", metas, isDefault)
}

func (m *Model) SetGroupMetadata(namespaces string, metas map[string]*GroupMetadata, isDefault bool) error {
    return m.setMetadata(s.prefix + "{groups." + namespaces + "}", metas, isDefault)
}

func (m *Model) SetUserMetadata(namespaces string, metas map[string]*UserMetadata, isDefault bool) error {
    return m.setMetadata(s.prefix + "{users." + namespaces + "}", metas, isDefault)
}

func (m *Model) DeleteNamespacesMetadata(namespaces []string) error {
    return m.delMetadata(s.predix + "{namespaces}", namespaces)
}

func (m *Model) DeleteGroupMetadata(namespace string, groups []string) error {
    return m.delMetadata(s.predix + "{groups." + namespace + "}", namespaces)
}

func (m *Model) DeleteUserMetadata(namespace string, users []string) error {
    return m.delMetadata(s.predix + "{users." + namespace + "}", namespaces)
}

func (m *Model) GetBlobMap(key string, timeout int, primitive BlobMapPersistPrimitive) *BlobMap {
    var ref *BlobMapReference

    newBlobMap := func() *BlobMap {
        if ref == nil {
            ref = &ModelBlobMapReference{
                Map: NewBlobMap(s.Pool, m.Prefix, key, timeout, primitive)
            }
            m.Log.Infof1("Add new blobmap \"" + key + "\"to pool.")
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

        ref, loaded = raw.(*BlobMap)
        if !loaded { // Wrong type. Force to replace.
            m.blobMapPool.Store(key, newBlobMap())
        }
        break
    }

    // update access time.
    ref.LastAccess = time.Now()

    return ref.Map
}
