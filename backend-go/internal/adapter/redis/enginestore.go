package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// EngineStore 用 Redis 实现 engine.Store（点击参数复用计数、交接 token、交接候选）。
type EngineStore struct{ rdb *goredis.Client }

func NewEngineStore(rdb *goredis.Client) *EngineStore { return &EngineStore{rdb: rdb} }

func (s *EngineStore) ClickSeen(ctx context.Context, siteID uint64, value, ipPrefix string, window time.Duration) (int, error) {
	key := fmt.Sprintf("clickip:%d:%s", siteID, value)
	pipe := s.rdb.Pipeline()
	pipe.SAdd(ctx, key, ipPrefix)
	pipe.Expire(ctx, key, window)
	card := pipe.SCard(ctx, key)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return int(card.Val()), nil
}

func (s *EngineStore) HandoffIssue(ctx context.Context, siteID uint64, tokenHash, attributionID, visitorID string, maxUses int, ttl time.Duration) error {
	key := fmt.Sprintf("hot:%d:%s", siteID, tokenHash)
	pipe := s.rdb.Pipeline()
	pipe.HSet(ctx, key, "aid", attributionID, "vid", visitorID, "max", maxUses, "used", 0)
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	return err
}

// handoffConsume 原子：存在且 used < max 时 used+1 并返回 aid、vid。
var handoffConsume = goredis.NewScript(`
local v = redis.call('HMGET', KEYS[1], 'aid', 'vid', 'max', 'used')
if not v[1] then return nil end
if tonumber(v[4]) >= tonumber(v[3]) then return nil end
redis.call('HINCRBY', KEYS[1], 'used', 1)
return {v[1], v[2]}
`)

func (s *EngineStore) HandoffConsume(ctx context.Context, siteID uint64, tokenHash string) (string, string, bool, error) {
	key := fmt.Sprintf("hot:%d:%s", siteID, tokenHash)
	res, err := handoffConsume.Run(ctx, s.rdb, []string{key}).Result()
	if err == goredis.Nil {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}
	arr, ok := res.([]interface{})
	if !ok || len(arr) != 2 {
		return "", "", false, nil
	}
	return fmt.Sprint(arr[0]), fmt.Sprint(arr[1]), true, nil
}

func (s *EngineStore) HandoffCandidateAdd(ctx context.Context, siteID uint64, ipHash, deviceFamily, attributionID, visitorID string, window time.Duration) error {
	key := fmt.Sprintf("hoc:%d:%s:%s", siteID, ipHash, deviceFamily)
	now := time.Now()
	pipe := s.rdb.Pipeline()
	pipe.ZAdd(ctx, key, goredis.Z{Score: float64(now.UnixMilli()), Member: attributionID + "|" + visitorID})
	pipe.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(now.Add(-window).UnixMilli(), 10))
	pipe.Expire(ctx, key, window)
	_, err := pipe.Exec(ctx)
	return err
}

// candidateTake 原子：取窗口内最近一条并删除。
var candidateTake = goredis.NewScript(`
local items = redis.call('ZREVRANGEBYSCORE', KEYS[1], '+inf', ARGV[1], 'LIMIT', 0, 1)
if #items == 0 then return nil end
redis.call('ZREM', KEYS[1], items[1])
return items[1]
`)

func (s *EngineStore) HandoffCandidateTake(ctx context.Context, siteID uint64, ipHash, deviceFamily string, window time.Duration) (string, string, bool, error) {
	key := fmt.Sprintf("hoc:%d:%s:%s", siteID, ipHash, deviceFamily)
	min := strconv.FormatInt(time.Now().Add(-window).UnixMilli(), 10)
	res, err := candidateTake.Run(ctx, s.rdb, []string{key}, min).Result()
	if err == goredis.Nil {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}
	str := fmt.Sprint(res)
	for i := 0; i < len(str); i++ {
		if str[i] == '|' {
			return str[:i], str[i+1:], true, nil
		}
	}
	return str, "", true, nil
}
