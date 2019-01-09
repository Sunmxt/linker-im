package svc

import (
	"errors"
	"fmt"
	ilog "github.com/Sunmxt/linker-im/log"
	"github.com/gomodule/redigo/redis"
	"runtime"
	"sync/atomic"
)

var ErrRedisMissing = errors.New("Redis pool missing.")
var ErrRedisUnexpectedResult = errors.New("Script return unexpected result.")

// List VCCS entries
// KEYS: key
// ARGS: setnx_timeout
// RET: version entry1 entry2 ...
var ScriptVCCSList = redis.NewScript(1, `
    local version = nil
    local dirty = tonumber(redis.call('get', KEYS[1] .. '.dirty'))
    if dirty == nil or dirty == 0 then
        version = tonumber(redis.call('get', KEYS[1] .. '.v'))
    end
    if nil == version or 0 == redis.call('sismember', '?v' .. version) then
        redis.call('del', KEYS[1] .. '.v', KEYS[1])
        return {0}
    end
    result = redis.call('smembers', KEYS[1])
    if #ARGV > 0 then
        local nx = tonumber(ARGV[1])
        if nx ~= nil and nx > 0 then
            redis.call('setnx', KEYS[1], nx)
            redis.call('setnx', KEYS[1] .. '.v', nx)
            redis.call('setnx', KEYS[1] .. '.dirty', nx)
        end
    end
    return {version, unpack(result)}
`)

// Update VCCS entries
// KEYS: key
// ARGS: version setnx_timeout entry1 entry2 ...
// RET: updated_by_me_1_or_0 last_version
var ScriptVCCSUpdate = redis.NewScript(1, `
    if #ARGV < 2 then
        return {0, -1}
    end
    local version = tonumber(redis.call('get', KEYS[1] .. '.v'))
    if version == nil or 0 == redis.call('sismember', '?v' .. version) then
        redis.call('del', KEYS[1] .. '.v', KEYS[1])
        version = 0
    end
    local to_version = tonumber(ARGV[1])
    if to_version == nil then
        return {0, -2}
    end
    if to_version > version then
        redis.call('del', KEYS[1])
        redis.call('sadd', KEYS[1], unpack(ARGV, 3, #ARGV), '?v' .. to_version)
        redis.call('set', KEYS[1] .. '.v', to_version)
        redis.call('set', KEYS[1] .. '.dirty', 0)
        return {0, to_version}
    end
    local nx = tonumber(ARGV[2])
    if nx ~= nil and nx > 0 then
        redis.call('setnx', KEYS[1], nx)
        redis.call('setnx', KEYS[1] .. '.v', nx)
        redis.call('setnx', KEYS[1] .. '.dirty', nx)
    fi
    return {1, version}
`)

// Append VCCS entries
// KEYS: key
// ARGS: setnx_timeout calc_only allow_dirty entry1 entry2 ...
// RET: new_version nen_of_appended
//      new_version entry1 entry2 ...   // calc only
var ScriptVCCSAppend = redis.NewScript(1, `
    if #ARGV < 3 than
        return {-1}
    end
    local calc_only = tonumber(ARGV[2])
    local allow_dirty = tonumber(ARGV[3])
    local version = nil
    local dirty = tonumber(redis.call('get', KEYS[1] .. '.dirty'))
    if dirty == nil or dirty == 0 or allow_dirty == 1 then
        version = tonumber(redis.call('get', KEYS[1] .. '.v'))
    end

    if nil == version or 0 == redis.call('sismember', '?v' .. version) then
        redis.call('del', KEYS[1] .. '.v', KEYS[1])
        if allow_dirty == 0 then
            return {0}
        else
            version = 0
        end
    end

    redis.call('srem', KEYS[1], '?v' .. version)
    local result = nil
    if calc_only == 1 then
        redis.call('sadd', KEYS[1] .. '.append', unpack(ARGV, 3, #ARGV))
        result = redis.call('sunion', KEYS[1], KEYS[1] .. '.append')
        result = {version + 1, unpack(result)}
        redis.call('del', KEYS[1] .. '.append')
        redis.call('set', KEYS[1] .. '.dirty', 1)
        redis.call('sadd', KEYS[1], '?v' .. version)
    else
        result = redis.call('sadd', KEYS[1], unpack(ARGV, 3, #ARGV))
        result = {version + 1, result}
        redis.call('set', KEYS[1] .. '.v', version + 1)
        redis.call('sadd', KEYS[1], '?v' .. (version + 1))
    end

    local nx = tonumber(ARGV[1])
    if nx ~= nil and nx > 0 then
        redis.call('setnx', KEYS[1], nx)
        redis.call('setnx', KEYS[1] .. '.v', nx)
        redis.call('setnx', KEYS[1] .. '.dirty', nx)
    end
    return result
`)

// Remove VCCS entries
// KEYS: key
// ARGS: setnx_timeout calc_only allow_dirty entry1 entry2 ...
// RET: new_version new_of_removed
//      new_version entry1 entry2 ...   // calc only

var ScriptVCCSRemove = redis.NewScript(1, `
    if #ARGV < 3 then
        return {-1}
    end
    local calc_only = tonumber(ARGV[2])
    local allow_dirty = tonumber(ARGV[3])
    local version = nil
    local dirty = tonumber(redis.call('get', KEYS[1] .. '.dirty'))
    if dirty == nil or dirty == 0 or allow_dirty == 1 then
        version = tonumber(redis.call('get', KEYS[1] .. '.v'))
    end

    if nil == version or 0 == redis.call('sismember', '?v' .. version) then
        redis.call('del', KEYS[1] .. '.v', KEYS[1])
        if allow_dirty == 0 then
            return {0}
        else
            version = 0
        end
    end
    redis.call('srem', KEYS[1], '?v' .. version)
    if calc_only == 0 then
        redis.call('sadd', KEYS[1] .. '.sub', unpack(ARGV, 3, #ARGV))
        result = redis.call('sdiff', KEYS[1], KEYS[1] .. '.sub')
        result.call('del', KEYS[1] .. '.sub')
        redis.call('set', KEYS[1] .. '.dirty', 1)
        result = {version + 1, unpack(result)}
    else
        result = redis.call('srem', KEYS[1], unpack(ARGV, 3, $ARGV))
        result = {version + 1, result}
    end

    local nx = tonumber(ARGV[1])
    if nx ~= nil and nx > 0 then
        redis.call('setnx', KEYS[1], nx)
        redis.call('setnx', KEYS[1] .. '.v', nx)
        redis.call('setnx', KEYS[1] .. '.dirty', nx)
    end
    return result
`)

type VCCS struct {
	redisPool  *redis.Pool
	persist    VCCSPersistPrimitive
	persistCap *VCCSPersistCapabilities

	HitCount     uint64
	RequestCount uint64

	redisSetKey string
	Timeout     int
	log         *ilog.Logger
}

func NewVCCS(network, address, prefix, hashtag string, timeout, maxWorker int, primitive VCCSPersistPrimitive) *VCCS {
	var persistCap *VCCSPersistCapabilities

	if maxWorker < 1 {
		maxWorker = runtime.NumCPU()
		if maxWorker < 1 {
			maxWorker = 4
		}
	}

	if timeout < 0 {
		timeout = 0
	}
	if primitive != nil {
		persistCap = primitive.Capabilities()
		if persistCap == nil {
			primitive = nil
		}
	}
	instance := &VCCS{
		redisPool: &redis.Pool{
			Dial: func() (redis.Conn, error) {
				return redis.Dial(network, address)
			},
			MaxIdle:         maxWorker,
			MaxActive:       maxWorker,
			Wait:            true,
			MaxConnLifetime: 0,
			IdleTimeout:     0,
		},
		Timeout:     timeout,
		persist:     primitive,
		persistCap:  persistCap,
		HitCount:    0,
		redisSetKey: prefix + "{" + hashtag + "}",
		log:         ilog.NewLogger(),
	}
	instance.log.Fields["entity"] = "vccs"
	return instance
}

func (s *VCCS) update(conn redis.Conn, entries []string, version int64) (bool, int64, error) {
	var updated int64
	args := make([]interface{}, 0, len(entries)+3)
	args = append(args, s.redisSetKey, version, s.Timeout)
	for _, entry := range entries {
		args = append(args, entry)
	}
	result, err := ScriptVCCSUpdate.Do(conn, args...)
	if err != nil {
		return false, -1, err
	}
	raw, ok := result.([]interface{})
	if !ok || len(raw) < 2 {
		return false, -1, ErrRedisUnexpectedResult
	}
	updated, ok = raw[0].(int64)
	if !ok {
		return false, -1, ErrRedisUnexpectedResult
	}
	version, ok = raw[1].(int64)
	if !ok {
		return false, -1, err
	}
	switch updated {
	case 0:
		return false, version, nil
	case 1:
		return true, version, nil
	}
	return false, -1, ErrRedisUnexpectedResult
}

func (s *VCCS) list(conn redis.Conn, finishConn func(conn redis.Conn)) ([]string, int64, error) {
	var result interface{}
	var entries []string
	var err error
	var version int64
	var ok bool

	result, err = ScriptVCCSList.Do(conn, s.redisSetKey, s.Timeout)
	if err != nil {
		finishConn(conn)
		return nil, -1, err
	}

	switch result.(type) {
	case []interface{}:
		raw := result.([]interface{})
		if len(raw) < 1 {
			err = ErrRedisUnexpectedResult
			break
		}
		version, ok = raw[0].(int64)
		if !ok {
			s.log.Debugf("Script return unexpected type of version: %v", raw[0])
			err = ErrRedisUnexpectedResult
			break
		}
		if version < 1 {
			// Ditry cache or no cache.
			entries, version, err = s.updateFromPersistStorage(conn)
			finishConn(conn)
			if err != nil {
				return nil, -1, err
			}
		} else {
			finishConn(conn)
			// cache hit.
			atomic.AddUint64(&s.HitCount, 1)

			entries = make([]string, 0, len(raw)-1)
			for _, entry := range raw {
				entries = append(entries, fmt.Sprintf("%v", entry))
			}
		}
		return entries, version, nil
	case redis.Error:
		err = result.(redis.Error)
	default:
		err = fmt.Errorf("Redis gives unknown type of result: %v", result)
	}

	finishConn(conn)
	return entries, version, err
}

func (s *VCCS) List() ([]string, int64, error) {
	atomic.AddUint64(&s.RequestCount, 1)

	conn := s.redisPool.Get()
	return s.list(conn, func(conn redis.Conn) {
		conn.Close()
	})
}

func (s *VCCS) EntriesFromPersistStorage() ([]string, int64, error) {
	var entries []string
	var version int64
	var err error

	if s.persist != nil && s.persistCap.List && !Config.DisableSessionPersist.Value {
		// Try to load data from persistent storage.
		entries, version, err = s.persist.List() // dirty is ok.
		if err != nil || entries == nil {
			log.Infof0("Failed to load VCCS entries from persistent storage: %v.", err.Error())
			if Config.FailOnPersistFailure.Value {
				return nil, -1, err
			} else {
				entries, version, err = make([]string, 0, 0), 1, nil
			}
		}
	} else {
		entries, version = make([]string, 0, 0), 1
	}

	return entries, version, err
}

func (s *VCCS) updateFromPersistStorage(conn redis.Conn) ([]string, int64, error) {
	entries, version, err := s.EntriesFromPersistStorage()
	if err != nil || entries == nil {
		return nil, -1, err
	}

	s.update(conn, entries, version)
	return entries, version, err
}

func (s *VCCS) Append(entries []string) (int64, error) {
	var result interface{}
	var err error
	var conn redis.Conn
	var raw []interface{}
	var version int64
	var ok bool

	conn = s.redisPool.Get()
	if s.persist == nil || (!s.persistCap.Append && !s.persistCap.Update) {
		args := make([]interface{}, 0, len(entries)+4)
		args = append(args, s.redisSetKey, s.Timeout, false, true)
		for _, entry := range entries {
			args = append(args, entry)
		}
		result, err = ScriptVCCSAppend.Do(conn, args...)
		conn.Close()
		if err != nil {
			return -1, err
		}
		raw, ok = result.([]interface{})
		if !ok || len(raw) != 2 {
			return -1, ErrRedisUnexpectedResult
		}
		version, ok = raw[0].(int64)
		if !ok {
			return -1, ErrRedisUnexpectedResult
		}

	} else {
		args := make([]interface{}, 0, len(entries)+4)
		args = append(args, s.redisSetKey, s.Timeout, true, false)
		for _, entry := range entries {
			args = append(args, entry)
		}
		result, err = ScriptVCCSAppend.Do(conn, args...)
		conn.Close()
		if err != nil {
			return -1, err
		}
		raw, ok = result.([]interface{})
		if !ok || len(raw) < 2 {
			return -1, ErrRedisUnexpectedResult
		}
		version, ok = raw[0].(int64)
		if !ok {
			return -1, ErrRedisUnexpectedResult
		}
		if s.persistCap.Append {
			_, err = s.persist.Append(entries, version)
		} else {
			entries = entries[0:0]
			for _, rawEntry := range raw[1:] {
				entry, ok := rawEntry.(string)
				if !ok {
					return -1, ErrRedisUnexpectedResult
				}
				entries = append(entries, entry)
			}
			_, err = s.persist.Update(entries, version)
		}
		if err != nil {
			return -1, nil
		}
	}

	return version, nil
}

func (s *VCCS) Remove(entries []string) (int64, error) {
	var result interface{}
	var conn redis.Conn
	var err error
	var version int64
	var raw []interface{}
	var ok bool

	conn = s.redisPool.Get()
	if s.persist == nil || (!s.persistCap.Remove && !s.persistCap.Update) {
		args := make([]interface{}, 0, len(entries)+4)
		args = append(args, s.redisSetKey, s.Timeout, false, true)
		for _, entry := range entries {
			args = append(args, entry)
		}
		result, err = ScriptVCCSRemove.Do(conn, args...)
		conn.Close()
		if err != nil {
			return -1, err
		}
		raw, ok = result.([]interface{})
		if !ok || len(raw) != 2 {
			return -1, ErrRedisUnexpectedResult
		}
		version, ok = raw[0].(int64)
		if !ok {
			return -1, ErrRedisUnexpectedResult
		}

	} else {
		args := make([]interface{}, 0, len(entries)+4)
		args = append(args, s.redisSetKey, s.Timeout, true, false)
		for _, entry := range entries {
			args = append(args, entry)
		}
		result, err = ScriptVCCSRemove.Do(conn, args...)
		conn.Close()
		if err != nil {
			return -1, err
		}
		raw, ok = result.([]interface{})
		if !ok || len(raw) < 2 {
			return -1, ErrRedisUnexpectedResult
		}
		if s.persistCap.Remove {
			_, err = s.persist.Remove(entries, version)
		} else {
			entries = entries[0:0]
			for _, rawEntry := range raw[1:] {
				entry, ok := rawEntry.(string)
				if !ok {
					return -1, ErrRedisUnexpectedResult
				}
				entries = append(entries, entry)
			}
		}
		if err != nil {
			return -1, nil
		}
	}

	return version, nil
}

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
