package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"cloak/internal/engine"
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

func (s *EngineStore) HandoffIssue(ctx context.Context, siteID uint64, tokenHash, attributionID, visitorID, fromRequestID string, maxUses int, ttl time.Duration) error {
	key := fmt.Sprintf("hot:%d:%s", siteID, tokenHash)
	pipe := s.rdb.Pipeline()
	pipe.HSet(ctx, key, "aid", attributionID, "vid", visitorID, "rid", fromRequestID, "max", maxUses, "used", 0)
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	return err
}

// handoffConsume 原子：存在且 used < max 时 used+1 并返回 aid、vid、rid。
var handoffConsume = goredis.NewScript(`
local v = redis.call('HMGET', KEYS[1], 'aid', 'vid', 'max', 'used', 'rid')
if not v[1] then return nil end
if tonumber(v[4]) >= tonumber(v[3]) then return nil end
redis.call('HINCRBY', KEYS[1], 'used', 1)
return {v[1], v[2], v[5] or ''}
`)

func (s *EngineStore) HandoffConsume(ctx context.Context, siteID uint64, tokenHash string) (engine.HandoffRef, bool, error) {
	key := fmt.Sprintf("hot:%d:%s", siteID, tokenHash)
	res, err := handoffConsume.Run(ctx, s.rdb, []string{key}).Result()
	if err == goredis.Nil {
		return engine.HandoffRef{}, false, nil
	}
	if err != nil {
		return engine.HandoffRef{}, false, err
	}
	arr, ok := res.([]interface{})
	if !ok || len(arr) < 2 {
		return engine.HandoffRef{}, false, nil
	}
	ref := engine.HandoffRef{AttributionID: fmt.Sprint(arr[0]), VisitorID: fmt.Sprint(arr[1])}
	if len(arr) > 2 && arr[2] != nil {
		ref.FromRequestID = fmt.Sprint(arr[2])
	}
	return ref, true, nil
}

func (s *EngineStore) HandoffCandidateAdd(ctx context.Context, siteID uint64, ipHash, deviceFamily string, ref engine.HandoffRef, window time.Duration) error {
	key := fmt.Sprintf("hoc:%d:%s:%s", siteID, ipHash, deviceFamily)
	now := time.Now()
	member := ref.AttributionID + "|" + ref.VisitorID + "|" + ref.FromRequestID
	pipe := s.rdb.Pipeline()
	pipe.ZAdd(ctx, key, goredis.Z{Score: float64(now.UnixMilli()), Member: member})
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

func (s *EngineStore) HandoffCandidateTake(ctx context.Context, siteID uint64, ipHash, deviceFamily string, window time.Duration) (engine.HandoffRef, bool, error) {
	key := fmt.Sprintf("hoc:%d:%s:%s", siteID, ipHash, deviceFamily)
	min := strconv.FormatInt(time.Now().Add(-window).UnixMilli(), 10)
	res, err := candidateTake.Run(ctx, s.rdb, []string{key}, min).Result()
	if err == goredis.Nil {
		return engine.HandoffRef{}, false, nil
	}
	if err != nil {
		return engine.HandoffRef{}, false, err
	}
	parts := strings.SplitN(fmt.Sprint(res), "|", 3)
	ref := engine.HandoffRef{AttributionID: parts[0]}
	if len(parts) > 1 {
		ref.VisitorID = parts[1]
	}
	if len(parts) > 2 {
		ref.FromRequestID = parts[2]
	}
	return ref, true, nil
}
