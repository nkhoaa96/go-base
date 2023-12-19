// Package cache is a simple cache wrapper, used to abstract Redis/Memcache/etc behind a reusable
// API for simple use cases.
//
// The idea is that Redis could be swapped for another cache and the client wouldn't
// need to update another (except perhaps calls to New to provide different connection
// parameters).
//
// For now cache supports only Redis, but eventually that could be provided by the client.
package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/nkhoaa96/go-base/storage/local"
	"strconv"
	"strings"
	"time"
)

const (
	lockScript = `
		return redis.call('SET', KEYS[1], ARGV[1], 'NX', 'PX', ARGV[2])
	`
	unlockScript = `
		if redis.call("get",KEYS[1]) == ARGV[1] then
		    return redis.call("del",KEYS[1])
		else
		    return 0
		end
	`
	// Retry mechanism, 2x by default
	retryTimes = 2

	ZPositiveInf = "+inf"
	ZNegativeInf = "-inf"
)

var (
	ErrNil               = redis.ErrNil
	ErrPoolExhausted     = redis.ErrPoolExhausted
	ErrConnClosed        = errors.New("redigo: connection closed")
	ErrCantUnlock        = errors.New("failed to unlock")
	ErrTTLNotSet         = errors.New("ttl is not set")
	ErrKeyNotExist       = errors.New("key does not exist")
	ErrDestinationNotSet = errors.New("destination is not set")
	ErrKeysNotSet        = errors.New("keys are not set")
	ErrExpireNotSet      = errors.New("key does not exist or the timeout could not be set")
)

// Cacher defines a mockable Cache interface that can store values in a key-value cache.
type Cacher interface {
	// Do sends a command to the server and returns the received reply.
	Do(cmd string, args ...interface{}) (reply interface{}, err error)

	PutString(key string, value string) (interface{}, error)
	GetString(key string) (string, error)

	PutMarshaled(key string, value interface{}) (interface{}, error)
	GetMarshaled(key string, v interface{}) error

	PutHead(key string, values ...string) (interface{}, error)
	PutTail(key string, values ...string) (interface{}, error)
	PopHead(key string, count int) (interface{}, error)
	PopTail(key string, count int) (interface{}, error)
	PutBefore(key string, value string, beforeValue string) (interface{}, error)
	PutAfter(key string, value string, afterValue string) (interface{}, error)
	GetAt(key string, index int) (interface{}, error)
	GetValues(key string, start, end int) ([]string, error)
	GetLength(key string) (int, error)
	DeleteValue(key string, value string, count int) (interface{}, error)

	Exist(keys ...string) (int, error)
	Delete(keys ...string) error
	Expire(key string, seconds time.Duration) error
	ExpireAt(key string, unixTimestamp int64) error

	Lock(key, value string, timeoutMs int) (bool, error)
	Unlock(key, value string) error

	Keys(pattern string) ([]string, error)
	Incr(key string) error
	TTL(key string) (int, error)
	IncrBy(key string, value interface{}) error
	DecrBy(key string, value interface{}) error

	HSet(key, field string, value interface{}) error
	HGet(key, field string, value interface{}) error
	HGetAll(key string) (map[string]string, error)
	HVals(key string, value interface{}) error
	HLen(key string) (int64, error)
	HMSet(key string, value map[string]interface{}) error
	HMGet(key string, fields []string, value interface{}) error
	HKeys(key string) ([]string, error)
	HDel(key string, fields []string) error
	HExists(key, field string) (bool, error)
	HMGetAsByteSlices(key string, fields []string) ([][]byte, error)

	ZAdd(key string, value *Z) error
	ZRem(key string, member ...string) error
	ZAdds(key string, value ...*Z) error
	ZCount(key string, min, max interface{}) (int64, error)
	ZPopMin(key string) ([]string, error)
	ZRevRangeByScore(key string, max, min, value interface{}) error
	ZRemRangeByScore(key string, min, max interface{}) error
}

// Cache implements the Cacher interface using a Redis pool.
type Cache struct {
	enableSSL bool
	pool      *redis.Pool
	//aiClient  appinsights.Client
}

type RedisConf struct {
	Server    string
	DB        int
	MaxIdle   int
	MaxActive int
}

type Z struct {
	Score  float64
	Member interface{}
}

// Option sets an optional parameter for Cache client.
type Option func(*Cache)

// New instantiates and returns a new Cache.
func New(address, password string, options ...Option) (Cacher, error) {
	var (
		maxIdle   = 2
		maxActive int
	)
	if v := local.Getenv("REDIS_MAX_IDLE"); v != "" {
		maxIdle, _ = strconv.Atoi(v)
	}
	if v := local.Getenv("REDIS_MAX_ACTIVE"); v != "" {
		maxActive, _ = strconv.Atoi(v)
	}
	var c = &Cache{}
	for _, option := range options {
		option(c)
	}

	// try to ping Redis
	conn, err := redis.Dial("tcp", address, redis.DialPassword(password), redis.DialUseTLS(c.enableSSL))
	if err != nil {
		//c.aiClient.TrackException(err)
		return nil, err
	}
	defer conn.Close()

	c.pool = &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", address, redis.DialPassword(password), redis.DialUseTLS(c.enableSSL))
		},
		MaxIdle:   maxIdle,
		MaxActive: maxActive,
		Wait:      true,
	}
	return c, nil
}

func EnableSSL(enableSSL bool) Option {
	return func(c *Cache) { c.enableSSL = enableSSL }
}

//// SetAIClient set App Insights client to Cache client.
//func SetAIClient(aiClient appinsights.Client) Option {
//	return func(c *Cache) { c.aiClient = aiClient }
//}

// PutString stores a simple key-value pair in the cache.
func (c *Cache) PutString(key string, value string) (interface{}, error) {
	return c.Do("SET", key, value)
}

// GetString returns the string value stored with the given key.
// If the key doesn't exist, an error is returned.
func (c *Cache) GetString(key string) (string, error) {
	return redis.String(c.Do("GET", key))
}

// PutMarshaled stores a json marshalled value with the given key.
func (c *Cache) PutMarshaled(key string, value interface{}) (interface{}, error) {
	// Marshal to JSON
	bytes, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	// Store in the cache
	return c.PutString(key, string(bytes[:]))
}

// GetMarshaled retrieves an item from the cache with the specified key,
// and un-marshals it from JSON to the value provided.
//
// If they key doesn't exist, an error is returned.
func (c *Cache) GetMarshaled(key string, v interface{}) error {
	cached, err := c.GetString(key)
	if err != nil {
		return err
	}
	if len(cached) > 0 {
		if err := json.Unmarshal([]byte(cached), v); err != nil {
			return err
		}
	}
	return nil
}

// PutHead inserts value to the head of array
func (c *Cache) PutHead(key string, values ...string) (interface{}, error) {
	args := []interface{}{key}
	for _, v := range values {
		args = append(args, v)
	}
	return c.Do("LPUSH", args...)
}

// PutTail inserts value to the tail of array
func (c *Cache) PutTail(key string, values ...string) (interface{}, error) {
	args := []interface{}{key}
	for _, v := range values {
		args = append(args, v)
	}
	return c.Do("RPUSH", args...)
}

// PopHead removes and returns the first element of the list stored at key
// When provided with the optional count argument, the reply will consist of up to count elements,
// depending on the list's length.
func (c *Cache) PopHead(key string, count int) (interface{}, error) {
	return c.Do("LPOP", key, count)
}

func (c *Cache) PopTail(key string, count int) (interface{}, error) {
	return c.Do("RPOP", key, count)
}

func (c *Cache) PutBefore(key string, value string, beforeValue string) (interface{}, error) {
	return c.Do("LINSERT", key, "BEFORE", beforeValue, value)
}

func (c *Cache) PutAfter(key string, value string, afterValue string) (interface{}, error) {
	return c.Do("LINSERT", key, "AFTER", afterValue, value)
}

func (c *Cache) GetAt(key string, index int) (interface{}, error) {
	return c.Do("LINDEX", key, index)
}

func (c *Cache) GetValues(key string, start, end int) ([]string, error) {
	return redis.Strings(c.Do("LRANGE", key, start, end))
}

func (c *Cache) GetLength(key string) (int, error) {
	return redis.Int(c.Do("LLEN", key))
}

func (c *Cache) DeleteValue(key string, value string, count int) (interface{}, error) {
	return redis.Int(c.Do("LREM", key, count, value))
}

func (c *Cache) Exist(keys ...string) (int, error) {
	var args []interface{}
	for _, key := range keys {
		args = append(args, key)
	}
	return redis.Int(c.Do("EXISTS", args...))
}

// Delete removes multiple keys
func (c *Cache) Delete(keys ...string) (err error) {
	var args []interface{}
	for _, key := range keys {
		args = append(args, key)
	}
	_, err = c.Do("DEL", args...)
	return
}

// Expire sets the time for a key to expire in seconds.
func (c *Cache) Expire(key string, timeout time.Duration) error {
	seconds := strconv.Itoa(int(timeout))
	reply, err := redis.Int(c.Do("EXPIRE", key, seconds))
	if err != nil {
		return err
	}
	if reply != 1 {
		return ErrExpireNotSet
	}
	return nil
}

func (c *Cache) ExpireAt(key string, unixTimestamp int64) error {
	reply, err := redis.Int(c.Do("EXPIREAT", key, unixTimestamp))
	if err != nil {
		return err
	}
	if reply != 1 {
		return ErrExpireNotSet
	}
	return nil
}

func (c *Cache) Keys(pattern string) ([]string, error) {
	return redis.Strings(c.Do("KEYS", pattern))
}

func (c *Cache) Incr(key string) error {
	_, err := c.Do("INCR", key)
	return err
}

func (c *Cache) IncrBy(key string, value interface{}) (err error) {
	_, err = c.Do("INCRBY", key, value)
	return
}

func (c *Cache) DecrBy(key string, value interface{}) (err error) {
	_, err = c.Do("DECRBY", key, value)
	return
}

func (c *Cache) TTL(key string) (int, error) {
	return redis.Int(c.Do("TTL", key))
}

// Lock attempts to put a lock on the key for a specified duration (in milliseconds).
// If the lock was successfully acquired, true will be returned.
//
// Note: The value provided can be anything, so long as it's unique. The value will then be used when
// attempting to Unlock, and will only work if the value matches. It's important that each instance that tries
// to perform a Lock have it's own unique key so that you don't unlock another instances lock!
func (c *Cache) Lock(key, value string, timeoutMs int) (bool, error) {
	r := c.pool.Get()
	defer func() {
		_ = r.Close()
	}()
	cmd := redis.NewScript(1, lockScript)
	res, err := cmd.Do(r, key, value, timeoutMs)
	if err != nil {
		return false, err
	}
	return res == "OK", nil
}

// Unlock attempts to remove the lock on a key so long as the value matches.
// If the lock cannot be removed, either because the key has already expired or
// because the value was incorrect, an error will be returned.
func (c *Cache) Unlock(key, value string) error {
	r := c.pool.Get()
	defer func() {
		_ = r.Close()
	}()
	cmd := redis.NewScript(1, unlockScript)
	if res, err := redis.Int(cmd.Do(r, key, value)); err != nil {
		return err
	} else if res != 1 {
		return ErrCantUnlock
	}
	return nil
}

func (c *Cache) ZAdd(key string, value *Z) error {
	bytes, err := json.Marshal(value.Member)
	if err != nil {
		return err
	}
	_, err = c.Do("ZADD", key, value.Score, string(bytes[:]))
	return err
}

func (c *Cache) ZRem(key string, member ...string) error {
	args := []interface{}{key}
	for _, v := range member {
		args = append(args, v)
	}
	_, err := c.Do("ZREM", args...)
	return err
}

func (c *Cache) ZAdds(key string, value ...*Z) error {
	args := []interface{}{key}
	for _, item := range value {
		bytes, err := json.Marshal(item.Member)
		if err != nil {
			return err
		}
		args = append(args, item.Score, string(bytes[:]))
	}
	_, err := c.Do("ZADD", args...)
	return err
}

func (c *Cache) ZCount(key string, min, max interface{}) (int64, error) {
	return redis.Int64(c.Do("ZCOUNT", key, min, max))
}

func (c *Cache) ZPopMin(key string) ([]string, error) {
	return redis.Strings(c.Do("ZPOPMIN", key))
}

func (c *Cache) ZRevRangeByScore(key string, max, min, value interface{}) error {
	slice, err := redis.Strings(c.Do("ZREVRANGEBYSCORE", key, max, min))
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(fmt.Sprintf("[%s]", strings.Join(slice, ","))), &value)
}

func (c *Cache) ZRemRangeByScore(key string, min, max interface{}) error {
	_, err := c.Do("ZREMRANGEBYSCORE", key, min, max)
	return err
}

func (c *Cache) HSet(key, field string, value interface{}) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = c.Do("HSET", key, field, bytes)
	return err
}

func (c *Cache) HGet(key, field string, value interface{}) error {
	bytes, err := redis.Bytes(c.Do("HGET", key, field))
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, &value)
}

func (c *Cache) HGetAll(key string) (map[string]string, error) {
	slice, err := redis.Strings(c.Do("HGETALL", key))
	if err != nil {
		return nil, err
	}
	var m = make(map[string]string)
	for i := 0; i < len(slice); i += 2 {
		m[slice[i]] = slice[i+1]
	}
	return m, nil
}

func (c *Cache) HLen(key string) (int64, error) {
	return redis.Int64(c.Do("HLEN", key))
}

func (c *Cache) HVals(key string, value interface{}) error {
	slice, err := redis.Strings(c.Do("HVALS", key))
	if err != nil {
		return err
	}
	var newSlice []string
	for _, val := range slice {
		if val != "" {
			newSlice = append(newSlice, val)
		}
	}
	return json.Unmarshal([]byte(fmt.Sprintf("[%s]", strings.Join(newSlice, ","))), &value)
}

func (c *Cache) HMSet(key string, value map[string]interface{}) error {
	args := []interface{}{key}
	for field, item := range value {
		bytes, err := json.Marshal(item)
		if err != nil {
			return err
		}
		args = append(args, field, bytes)
	}
	_, err := c.Do("HMSET", args...)
	return err
}

func (c *Cache) HMGet(key string, fields []string, value interface{}) error {
	args := []interface{}{key}
	for _, field := range fields {
		args = append(args, field)
	}
	slice, err := redis.Strings(c.Do("HMGET", args...))
	if err != nil {
		return err
	}
	var newSlice []string
	for _, val := range slice {
		if val != "" {
			newSlice = append(newSlice, val)
		}
	}
	return json.Unmarshal([]byte(fmt.Sprintf("[%s]", strings.Join(newSlice, ","))), &value)
}

func (c *Cache) HExists(key, field string) (bool, error) {
	return redis.Bool(c.Do("HEXISTS", key, field))
}

func (c *Cache) HMGetAsByteSlices(key string, fields []string) ([][]byte, error) {
	args := []interface{}{key}
	for _, field := range fields {
		args = append(args, field)
	}
	return redis.ByteSlices(c.Do("HMGET", args...))
}

func (c *Cache) HKeys(key string) ([]string, error) {
	return redis.Strings(c.Do("HKEYS", key))
}

func (c *Cache) HDel(key string, fields []string) error {
	args := []interface{}{key}
	for _, field := range fields {
		args = append(args, field)
	}
	_, err := c.Do("HDEL", args...)
	return err
}

func (c *Cache) Do(cmd string, args ...interface{}) (reply interface{}, err error) {
	var (
		r     = c.pool.Get()
		retry = 1
	)
	defer r.Close()

	//c.trackException(r.Err())

	for retry <= retryTimes {
		if reply, err = r.Do(cmd, args...); err == nil || errors.Is(err, ErrNil) {
			return
		}
		retry++
	}
	return
}

//func (c *Cache) trackException(err error) {
//	if err == nil || c.aiClient == nil {
//		return
//	}
//	c.aiClient.TrackException(err)
//}
