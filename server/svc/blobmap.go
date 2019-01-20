package svc


import (
    "errors"
    "fmt"
    "github.com/gomodule/redigo/redis"

    ilog "github.com/Sunmxt/linker-im/log"
)


type BlobMap struct {
    RedisPool   *redis.Pool
    persist BlobMapPersistPrimitive
    prefix string
    tag string
    timeout int

    Log *ilog.Logger
}

// Errors
var ErrInsurfficientValues = errors.New("Insurfficient values returned from redis.")
var ErrInvalidResult =  errors.New("Invalid result returned from redis.")
var ErrInvalidVersionType = errors.New("Invalid data type of \"version\".")

// Gets
// Key: key
// Args: allow_dirty entry1 [entry2 ...]
// RET: version value1 [value2 ...]
//      0
var ScriptBlobMapGets = redis.NewScript(1, `
    local version = tonumber(redis.call('HGET', KEYS[1], '#?v'))
    local dirty = tonumber(redis.call('GET', KEYS[1] .. '.d'))
    local allow_dirty = tonumber(ARGV[1])
    if version == nil or version < 1 or dirty == 1 then
        if allow_dirty ~= 1 then
            return 0
        else
            version = 0
        end
    end
    local result = {version}
    if #ARGV > 1 then
        result = {unpack(result), unpack(redis.call('HMGET', KEYS[1], unpack(ARGV, 2, #ARGV)))}
    end
    return result
`)

// Get all keys
// Key: key
// Args: allow_dirty
// RET: version key1 [key2 ...]
var ScriptBlobMapKeys = redis.NewScript(1, `
    local version = tonumber(redis.call('HGET', KEYS[1], '#?v'))
    local dirty = tonumber(redis.call('GET', KEYS[1] .. '.d'))
    local allow_dirty = tonumber(ARGV[1])
    if version == nil or version < 1 or dirty == 1 then
        if allow_dirty ~= 1 then
            return 0
        else
            version = 0
        end
    end
    local result = {version}
    result = {unpack(result), unpack(redis.call('HKEYS', KEYS[1]))}
    return result
`)

// New version
// Key: key
// Args: allow_dirty
// RET: new_version
var ScriptBlobMapNewVersion = redis.NewScript(1, `
    local allow_dirty = tonumber(ARGV[1])
    local version = tonumber(redis.call('HGET', KEYS[1], '#?v'))
    if version == nil or version < 1 then
        if allow_dirty == 1 then
            version = 1
        else
            return 0
        end
    end
    local new_version = tonumber(redis.call('GET', KEYS[1] .. '.av'))
    if new_version == nil or version > new_version then
        new_version = version
    end
    new_version = new_version + 1
    redis.call('SET', KEYS[1] .. '.av', new_version)
    return new_version
`)

// Update (never used when persist enabled.)
// Key: key
// Args: new_version [field1 key1 ...]
// Ret: version
var ScriptBlobMapUpdate(1, `
    local version = tonumber(redis.call('HGET', KEYS[1], '#?v'))
    local new_version = tonumber(ARGV[1])
    local result = version
    if new_version == nil then
        new_version = 1
    end
    if version == nil or version < 0 or new_version > version nthen
        redis.call('HSET', KEYS[1], '#?v', new_version)
        result = new_version
    end
    if #ARGV > 1 then
        redis.call('HMSET', KEYS[1], unpack(ARGV, 1, #ARGV))
    end
    return result
`)

// replace
// Keys: key
// Args: version allow_dirty [field1 value1 ...]
// Ret: new_version replaced
var ScriptBlobMapReplace(1, `
    local version = tonumber(redis.call('HGET', KEYS[1], '#?v'))
    local dirty = tonumber(redis.call('GET', KEYS[1] .. '.d'))
    local allow_dirty = tonumber(ARGV[2])
    local new_version = tonumber(ARGV[1])
    local result = nil
    if new_version == nil then
        new_version = 0
    end
    if version == nil or version < 1 then
        redis.call('DEL', KEYS[1])
        version = 0
        if new_version < 1 then
            new_version = 1
        end
    end
    if new_version > version or allow_dirty == 1 then
        redis.call('DEL', KEYS[1])
        if #ARGV > 2 then
            redis.call('HMSET', KEYS[1], unpack(ARGV, 3, #ARGV))
        end
        redis.call('SET', KEYS[1] .. '.d', 0)
        redis.call('HSET', KEYS[1], '#?v', new_version)
        result = {new_version, 1}
    else
        result = {version, 0}
    end
    return result
`)

type BlobMapPersistPrimitive interface {
    Loads(tag string) (map[string]string, int64, error)
    Sets(tag string, kv map[string]string, version int64) error
}

func NewBlobMap(redisPool *redis.Pool, prefix, tag string, timeout int, primitive BlobMapPersistPrimitive) *BlobMap {
    instance := &BlobMap{
        RedisPool: redisPool,
        persist: primitive,
        prefix: prefix,
        tag: tag,
        timeout: timeout,
        Log: ilog.NewLogger(),
    }

    instance.Log.Fields["entity"] = "blob_map"
    return instance
}


func (b *BlobMap) loadBlobs(conn redis.Conn, allowDirty bool) (map[string]string, int64, error) {
    kv, version, err := b.persist.Loads(b.tag)
    if err != nil {
        b.Log.Error("BlobMap persist.Load raise an error : " + err.Error())
        return nil, 0, err
    }
    _, err = b.replace(conn, kv, version, allowDirty)
    if err != nil {
        b.Log.Error("BlobMap replace failure (" + err.Error() + ")")
        return 
    }
    return kv, version, nil
}

func (b *BlobMap) gets(conn redis.Conn, keys []string, allowDirty bool) ([]string, int64, error) {
    var values []string
    var version int64

    args := make([]interface{}, 0, len(keys)+3)
    args = append(args, b.prefix + "{" + tag + "}", b.timeout, allowDirty)
    for key := range keys {
        args = append(args, key)
    }
    result, err := ScriptBlobMapGets.Do(conn, args...)
    if err != nil {
        return nil, 0, err
    }
    switch raw := result.(type) {
    case int64:
        return nil, raw, nil
    case []interface{}:
        if len(law) != len(keys) + 1 {
            return nil, 0, ErrInsurfficientValues
        }
        if version, err = redis.Int64(raw[0], ErrInvalidVersionType); err != nil {
            return nil, 0, err
        }
        if values, err = redis.Strings(raw[1:], errors.New("Invalid data type of values")) ; err != nil {
            return nil, version, err
        }
        return values, version, nil
    }

    return nil, 0, ErrInvalidResult
}

func (b *BlobMap) keys(conn redis.Conn, allowDirty bool) ([]string, int64, error) {
    var keys []string
    var raw []interface{}
    var version int64

    result, err := ScriptBlobMapKeys.Do(conn, b.prefix + "{" + tag + "}", b.timeout, allowDirty)
    if err != nil {
        return nil, 0, err
    }
    if raw, err = redis.Values(result, ErrInvalidResult); err != nil {
        return nil, 0, err
    }
    if len(raw) < 1 {
        return nil, 0, ErrInsurfficientValues
    }
    version, err = redis.Int64(raw[0], ErrInvalidVersionType)
    if err != nil {
        return nil, 0, err
    }
    if keys, err = redis.Strings(raw[1:], ErrInvalidResult); err != nil {
        return nil, version, err
    }
    return keys, version, nil
}

func (b *BlobMap) replace(conn redis.Conn, kv map[string]string, version int64, allowDirty bool) (int64, error) {
    var raw []interface{}
    var version int64
    args := make([]interface{}, 0, len(kv) * 2 + 2)
    args = append(args, b.timeout, version, allowDirty)
    for k, v := range kv {
        args = append(args, k, v)
    }
    result, err := ScriptBlobMapReplace.Do(conn, args...)
    if err != nil {
        return 0, err
    }
    raw, err = redis.Values(result, ErrInvalidResult)
    if len(raw) < 2 {
        return 0, ErrInsurfficientValues
    }
    version, err = redis.Int64(raw[0], ErrInvalidVersionType)
    return version, nil
}

func (b *BlobMap) update(conn redis.Conn, kv map[string]string, version int64) error {
    args := make()
    ScriptBlobMapUpdate.Do(conn,)
}

func (b *BlobMap) updatTimeout(conn redis.Conn) error {
    var err error
    if b.timeout < 1 {
        return nil
    }

    if err = conn.Send('SETNX', b.prefix + "{" + tag + "}", b.timeout); err != nil {
        return err
    }
    if err = conn.Send('SETNX', b.prefix + "{" + tag + "}.d", b.timeout); err != nil {
        return err
    }
    if err = conn.Flush(); err != nil {
        return err
    }
    for i := 2; i > 0 ; i -- {
        if err = conn.Receive(); err != nil {
            return err
        }
    }
    return nil
}

func (b *BlobMap) newVersion(conn redis.Conn, allowDirty bool) (int64, error) {
    var version int64

    for {
        result, err := ScriptBlobMapNewVersion.Do(conn, b.persist + "{" + tag + "}", allowDirty)
        if version, err = redis.Int64(result, ErrInvalidVersionType); err != nil {
            return 0, err
        }
        if allowDirty {
            break
        }
        if version < 1 {
            if _, _, err = b.loadBlobs(conn, false); err != nil {
                return 0, err
            }
            allowDirty = false
            continue
        }
        break
    }

    return version, nil
}

func (b *BlobMap) Keys() ([]string, int64, error) {
    conn := b.RedisPool.Get()
    defer conn.Close()

    persistEnabled := b.persist != nil
    values, version, err := b.keys(conn)
    if err != nil {
        return nil, err
    }
    if values == nil {
        values = make([]string)
    }
    if version < 1 && persistEnabled {
        var kv map[string]string
        if kv, version, err = loadBlobs(conn, false); err != nil {
            return nil, 0, err
        }
        for k, _ := range kv {
            values = append(values, k)
        }
    }
    b.updatTimeout(conn)
    return values, version, nil
}

func (b *BlobMap) Gets(keys []string) ([]string, int64, error) {
    conn := b.RedisPool.Get()
    defer conn.Close()

    persistEnabled := b.persist != nil
    values, version, err := b.gets(conn, keys, !persistEnabled)
    if err != nil  {
        return nil, err
    }
    if values == nil {
        values = make([]string, len(keys), len(keys))
    }
    if version < 1 && persistEnabled {
        var kv map[string]string
        if kv, version, err = loadBlobs(conn, false); err != nil {
            return nil, err
        }
        for idx, key := range keys {
            value, ok := kv[key]
            if !ok {
                value = ""
            }
            values[idx] = value
        }
    }

    b.updatTimeout(conn)
    return values, nil
}

func (b *BlobMap) Sets(kv map[string]string) (int64, error) {
    conn := b.RedisPool.Get()
    defer conn.Close()

    persistEnabled := b.persist != nil
    version, err := b.newVersion(conn, !persistEnabled)
    if err != nil {
        return version, err
    }
    if persistEnabled {
        if err = b.persist.Sets(b.tag, kv, version); err != nil {
            return version, err
        }
        _, err = conn.Do("SET", b.persist + "{" + b.tag + "}.d", 1)
    } else {
        err = b.update(kv, version)
    }
    if err != nil {
        return version, err
    }

    b.updatTimeout(conn)
    return version, nil
}
