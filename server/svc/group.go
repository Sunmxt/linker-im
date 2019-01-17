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

type groupVCCSInformation struct {
    set *VCCS
    LastAccess time.Time
}

type SessionGroup struct {
    vccs sync.Map
    prefix string
    timeout int
    primitive VCCSPersistPrimitive
    pool *redis.Pool
	log *ilog.Logger
}

func NewSessionGroup(redisPool *redis.Pool, prefix string, timeout int, primitive VCCSPersistPrimitive) *SessionGroup {
    instance := &SessionGroup{
        log: ilog.NewLogger(),
        prefix: prefix,
        timeout: timeout,
    }
    log.Fields["entity"] = "session-group"
}


func (g *SessionGroup) getVCCS(namespace string) *VCCS {
    var gInfo *groupVCCSInformation

    newVCCS := func() {
        if gInfo == nil {
            gInfo = &SessionGroup{
                set: NewVCCS(g.pool, prefix, tagName, timeout, primitive),
            }
        }

        return gInfo
    }

    tagName := "session-group-" + namespace
    for gInfo == nil {
        raw, loaded := g.vccs.Load(tagName)

        if !loaded {
            raw, loaded = g.vccs.LoadOrStore(tagName, newVCCS())
            if !loaded { // Stored by me.
                break
            }
        }

        gInfo, loaded = raw.(*groupVCCSInformation)
        if !loaded { // Wrong type. Force to replace.
            g.vccs.Store(tagName, newVCCS())
        }
        break
    }

    // update access time.
    gInfo.LastAccess = time.Now()

    return gInfo.set
}

// Allocate an unique ID and append it to group in specified namespace.
func (g *SessionGroup) Append(namespace string) proto.ID {
    var newID proto.ID

    for {
        newID = proto.NewID()
        vccs := g.getVCCS(namespace)
        key := VCCSEscapeKey(newID.AsKey())
        appendc, version, err := vccs.Append([]string{key})

        if err != nil {
            g.log.Error("Group appending failure: " + err.Error())
            return proto.EMPTY_ID
        }

        if appendc > 0 {
            g.log.Info0("Group \"%v\" added to namespace \"%v\". (version = %v)", newID.String(), namespace, version)
            break
        }
    }

    return newID
}

// Remove groups in specified namespace.
func (g *SessionGroup) Remove(namespace string, groups []proto.ID) uint {
    escaped := make([]string, 0, len(groups))
    for _, id := range groups {
        escaped = append(escaped, VCCSEscapeKey(id.AsKey()))
    }

    vccs := g.getVCCS(namespace)

    removec, version, err := vccs.Remove(escaped)
    if err != nil {
        g.log.Error("Group removal failure: " + err.Error())
        return 0
    }
    escaped = escaped[0:0]
    for _, id := range groups {
        escaped = append(escaped, id.String())
    }

    g.log.Info0("%v of groups \"%v\" removed in namespace \"%v\". (version = %v)", removec, strings.Join(escaped, "\",\""), namespace, version)

    return removec
}

// List groups in specified namespace.
func (g *SessionGroup) List(namespace string) []proto.ID {
    var id proto.ID
    vccs := g.getVCCS(namespace)

    raws, version, err := vccs.List()
    if err != nil {
        g.log.Error("Group listing failure: " + err.Error())
        return make([]proto.ID, 0, 0)
    }

    groups := make([]proto.ID, 0, len(raws))
    for _, raw := range raws {
        err = id.FromKey(VCCSUnescapeKey(raw))
        if err != nil {
            g.log.Warn("Unexcepted group ID found: " + err.Error())
            continue
        }
        groups = append(groups, id)
    }

    return groups
}

// Optimize clean idle VCCS instance.
func (g *SessionGroup) Optimize(before time.Time) {
    g.vccs.Range(func (key, value interface{}) bool {
        gInfo, ok := value.(*groupVCCSInformation)
        if !ok || gInfo.LastAccess.Before(before) {
            g.vccs.Delete(key)
        }
    })
}
