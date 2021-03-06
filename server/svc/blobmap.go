package svc

import (
	"errors"
	"github.com/gomodule/redigo/redis"

	ilog "github.com/Sunmxt/linker-im/log"
)

type BlobMap struct {
	RedisPool *redis.Pool
	persist   BlobMapPersistPrimitive
	prefix    string
	tag       string
	Timeout   int

	Log *ilog.Logger
}

// Errors
var ErrInsurfficientValues = errors.New("Insurfficient values returned from redis.")
var ErrInvalidResult = errors.New("Invalid result returned from redis.")
var ErrInvalidParams = errors.New("Invalid parameters.")
var ErrInvalidWriteOperation = errors.New("Invalid write operation.")

const (
	OP_SET         = 1
	OP_SET_DEFAULT = 2
	OP_DEL         = 3
)

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
    redis.call('HDEL', KEYS[1], '#?v')
    if #ARGV > 1 then
        result = {unpack(result), unpack(redis.call('HMGET', KEYS[1], unpack(ARGV, 2, #ARGV)))}
    end
    redis.call('HSET', KEYS[1], '#?v', version)
    return result
`)

// Get all keys
// Key: key
// Args: allow_dirty
// RET: version [key1 key2 ...]
var ScriptBlobMapKeys = redis.NewScript(1, `
    local version = tonumber(redis.call('HGET', KEYS[1], '#?v'))
    local dirty = tonumber(redis.call('GET', KEYS[1] .. '.d'))
    local allow_dirty = tonumber(ARGV[1])
    if version == nil or version < 1 or dirty == 1 then
        if allow_dirty ~= 1 then
            return {0}
        else
            version = 0
        end
    end
    local result = {version}
    redis.call('HDEL', KEYS[1], '#?v')
    result = {unpack(result), unpack(redis.call('HKEYS', KEYS[1]))}
    redis.call('HSET', KEYS[1], '#?v', version)
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
// Args: op new_version [field1 key1 ...]
// Ret: version
var ScriptBlobMapUpdate = redis.NewScript(1, `
    local version = tonumber(redis.call('HGET', KEYS[1], '#?v'))
    local new_version = tonumber(ARGV[2])
    local result = version
    local op = tonumber(ARGV[1])
    if new_version == nil then
        new_version = 1
    end
    if version == nil or version < 0 or new_version > version then
        redis.call('HSET', KEYS[1], '#?v', new_version)
        result = new_version
    end
    if #ARGV > 2 then
        if op == 2 then
            for i = 4, #ARGV, 2 do
                redis.call('HSETNX', KEYS[1], ARGV[i - 1], ARGV[i])
            end 
        elseif op == 1 then
            redis.call('HMSET', KEYS[1], unpack(ARGV, 3, #ARGV))
        elseif op == 3 then
            redis.call('HDEL', KEYS[1], unpack(ARGV, 3, #ARGV))
        end
    end
    return result
`)

// replace
// Keys: key
// Args: version allow_dirty [field1 value1 ...]
// Ret: new_version replaced
var ScriptBlobMapReplace = redis.NewScript(1, `
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

func NewBlobMap(redisPool *redis.Pool, prefix, tag string, timeout int, primitive BlobMapPersistPrimitive) *BlobMap {
	instance := &BlobMap{
		RedisPool: redisPool,
		persist:   primitive,
		prefix:    prefix,
		tag:       tag,
		Timeout:   timeout,
		Log:       ilog.NewLogger(),
	}

	instance.Log.Fields["entity"] = "blob_map"
	return instance
}

func (b *BlobMap) loadBlobs(conn redis.Conn, allowDirty bool) (map[string][]byte, int64, error) {
	kv, version, err := b.persist.Loads(b.tag)
	if err != nil {
		b.Log.Error("BlobMap persist.Load raise an error : " + err.Error())
		return nil, 0, err
	}
	_, err = b.replace(conn, kv, version, allowDirty)
	if err != nil {
		b.Log.Error("BlobMap replace failure (" + err.Error() + ")")
		return kv, version, err
	}
	return kv, version, nil
}

func (b *BlobMap) gets(conn redis.Conn, keys []string, allowDirty bool) ([][]byte, int64, error) {
	var values [][]byte
	var version int64

	args := make([]interface{}, 0, len(keys)+3)
	args = append(args, b.prefix+"{"+b.tag+"}", b.Timeout, allowDirty)
	for _, key := range keys {
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
		if len(raw) != len(keys)+1 {
			return nil, 0, ErrInsurfficientValues
		}
		if version, err = redis.Int64(raw[0], nil); err != nil {
			return nil, 0, err
		}
		if values, err = redis.ByteSlices(raw[1:], nil); err != nil {
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

	result, err := ScriptBlobMapKeys.Do(conn, b.prefix+"{"+b.tag+"}", b.Timeout, allowDirty)
	if err != nil {
		return nil, 0, err
	}
	if raw, err = redis.Values(result, nil); err != nil {
		return nil, 0, err
	}
	if len(raw) < 1 {
		return nil, 0, ErrInsurfficientValues
	}
	version, err = redis.Int64(raw[0], nil)
	if err != nil {
		return nil, 0, err
	}
	if keys, err = redis.Strings(raw[1:], nil); err != nil {
		return nil, version, err
	}
	return keys, version, nil
}

func (b *BlobMap) replace(conn redis.Conn, kv map[string][]byte, version int64, allowDirty bool) (int64, error) {
	var raw []interface{}
	args := make([]interface{}, 0, len(kv)*2+2)
	args = append(args, b.prefix+"{"+b.tag+"}", version, allowDirty)
	for k, v := range kv {
		args = append(args, k, v)
	}
	result, err := ScriptBlobMapReplace.Do(conn, args...)
	if err != nil {
		return 0, err
	}
	raw, err = redis.Values(result, nil)
	if len(raw) < 2 {
		return 0, ErrInsurfficientValues
	}
	version, err = redis.Int64(raw[0], nil)
	return version, nil
}

func (b *BlobMap) update(conn redis.Conn, raw interface{}, version int64, op int) error {
	var args []interface{}
	if op == OP_SET || op == OP_SET_DEFAULT {
		kv, ok := raw.(map[string][]byte)
		args = make([]interface{}, 0, len(kv)*2+3)
		args = append(args, b.prefix+"{"+b.tag+"}", op, version)
		if !ok {
			return ErrInvalidParams
		}
		for k, v := range kv {
			args = append(args, k, v)
		}
	} else if op == OP_DEL {
		keys, ok := raw.([]string)
		args = make([]interface{}, 0, len(keys)+3)
		args = append(args, b.prefix+"{"+b.tag+"}", op, version)
		if !ok {
			return ErrInvalidParams
		}
		for _, key := range keys {
			args = append(args, key)
		}
	} else {
		return ErrInvalidWriteOperation
	}
	_, err := ScriptBlobMapUpdate.Do(conn, args...)
	if err != nil {
		return err
	}
	return nil
}

func (b *BlobMap) updatTimeout(conn redis.Conn) error {
	var err error
	if b.Timeout < 1 {
		return nil
	}

	if err = conn.Send("SETNX", b.prefix+"{"+b.tag+"}", b.Timeout); err != nil {
		return err
	}
	if err = conn.Send("SETNX", b.prefix+"{"+b.tag+"}.d", b.Timeout); err != nil {
		return err
	}
	if err = conn.Flush(); err != nil {
		return err
	}
	for i := 2; i > 0; i-- {
		if _, err = conn.Receive(); err != nil {
			return err
		}
	}
	return nil
}

func (b *BlobMap) Destroy() error {
	return errors.New("Not supported")
}

func (b *BlobMap) newVersion(conn redis.Conn, allowDirty bool) (int64, error) {
	var version int64

	for {
		result, err := ScriptBlobMapNewVersion.Do(conn, b.prefix+"{"+b.tag+"}", allowDirty)
		if version, err = redis.Int64(result, nil); err != nil {
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
	values, version, err := b.keys(conn, !persistEnabled)
	if err != nil {
		return nil, 0, err
	}
	if values == nil {
		values = make([]string, 0)
	}
	if version < 1 && persistEnabled {
		var kv map[string][]byte
		if kv, version, err = b.loadBlobs(conn, false); err != nil {
			return nil, 0, err
		}
		for k, _ := range kv {
			values = append(values, k)
		}
	}
	b.updatTimeout(conn)
	return values, version, nil
}

func (b *BlobMap) Gets(keys []string) ([][]byte, int64, error) {
	conn := b.RedisPool.Get()
	defer conn.Close()

	persistEnabled := b.persist != nil
	values, version, err := b.gets(conn, keys, !persistEnabled)
	if err != nil {
		return nil, 0, err
	}
	if values == nil {
		values = make([][]byte, len(keys), len(keys))
	}
	if version < 1 && persistEnabled {
		var kv map[string][]byte
		if kv, version, err = b.loadBlobs(conn, false); err != nil {
			return nil, 0, err
		}
		for idx, key := range keys {
			value, ok := kv[key]
			if !ok {
				value = nil
			}
			values[idx] = value
		}
	}

	b.updatTimeout(conn)
	return values, version, nil
}

func (b *BlobMap) writeOp(kvRaw interface{}, op int) (int64, error) {
	conn := b.RedisPool.Get()
	defer conn.Close()

	persistEnabled := b.persist != nil
	version, err := b.newVersion(conn, !persistEnabled)
	if err != nil {
		return version, err
	}
	if persistEnabled {
		switch op {
		case OP_SET:
			kv, ok := kvRaw.(map[string][]byte)
			if !ok {
				return version, ErrInvalidParams
			}
			err = b.persist.Sets(b.tag, kv, version)
		case OP_SET_DEFAULT:
			kv, ok := kvRaw.(map[string][]byte)
			if !ok {
				return version, ErrInvalidParams
			}
			err = b.persist.SetDefaults(b.tag, kv, version)
		case OP_DEL:
			v, ok := kvRaw.([]string)
			if !ok {
				return version, ErrInvalidParams
			}
			err = b.persist.Dels(b.tag, v, version)
		default:
			return version, ErrInvalidWriteOperation
		}
		if err != nil {
			return version, err
		}
		_, err = conn.Do("SET", b.prefix+"{"+b.tag+"}.d", 1)
	} else {
		err = b.update(conn, kvRaw, version, op)
	}

	if err != nil {
		return version, err
	}
	b.updatTimeout(conn)
	return version, nil
}

func (b *BlobMap) SetDefaults(kv map[string][]byte) (int64, error) {
	return b.writeOp(kv, OP_SET_DEFAULT)
}

func (b *BlobMap) Sets(kv map[string][]byte) (int64, error) {
	return b.writeOp(kv, OP_SET)
}

func (b *BlobMap) Dels(keys []string) (int64, error) {
	return b.writeOp(keys, OP_DEL)
}
