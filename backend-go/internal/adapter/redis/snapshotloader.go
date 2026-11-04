package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"cloak/internal/engine/snapshot"
)

// 与 snapshotsvc 约定一致的 key；这里重复定义避免 adapter 依赖 app 层。
const (
	keySitePrefix = "cfg:site:"
	keyHostPrefix = "host:"
	chanChanged   = "cfg:changed"
)

// SnapshotLoader 实现 snapshot.Loader。
type SnapshotLoader struct{ rdb *goredis.Client }

func NewSnapshotLoader(rdb *goredis.Client) *SnapshotLoader { return &SnapshotLoader{rdb: rdb} }

func (l *SnapshotLoader) LoadSite(ctx context.Context, siteID uint64) (*snapshot.Site, error) {
	raw, err := l.rdb.Get(ctx, keySitePrefix+fmt.Sprint(siteID)).Bytes()
	if err != nil {
		return nil, err
	}
	var s snapshot.Site
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	if err := s.Compile(); err != nil {
		return nil, err
	}
	return &s, nil
}

func (l *SnapshotLoader) ResolveHost(ctx context.Context, host string) (uint64, error) {
	v, err := l.rdb.Get(ctx, keyHostPrefix+strings.ToLower(host)).Result()
	if err != nil {
		return 0, err
	}
	var id uint64
	_, err = fmt.Sscanf(v, "%d", &id)
	return id, err
}

// ResolvePrefix 服务端密钥前缀 → 站点 ID。
func (l *SnapshotLoader) ResolvePrefix(ctx context.Context, prefix string) (uint64, error) {
	v, err := l.rdb.Get(ctx, "sk:"+prefix).Result()
	if err != nil {
		return 0, err
	}
	var id uint64
	_, err = fmt.Sscanf(v, "%d", &id)
	return id, err
}

// Subscribe 订阅配置变更，收到后刷新或删除对应站点。阻塞直到 ctx 结束。
func (l *SnapshotLoader) Subscribe(ctx context.Context, store *snapshot.Store, log *zap.Logger) {
	sub := l.rdb.Subscribe(ctx, chanChanged)
	defer sub.Close()
	ch := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			id, deleted, valid := parseChange(msg.Payload)
			if !valid {
				continue
			}
			if deleted {
				store.Remove(id)
				log.Info("snapshot removed", zap.Uint64("site", id))
				continue
			}
			if store.Refresh(ctx, id) {
				log.Info("snapshot refreshed", zap.Uint64("site", id))
			} else {
				log.Warn("snapshot refresh failed", zap.Uint64("site", id))
			}
		}
	}
}

func parseChange(msg string) (uint64, bool, bool) {
	if !strings.HasPrefix(msg, "site:") {
		return 0, false, false
	}
	rest := strings.TrimPrefix(msg, "site:")
	deleted := strings.HasSuffix(rest, ":deleted")
	rest = strings.TrimSuffix(rest, ":deleted")
	var id uint64
	if _, err := fmt.Sscanf(rest, "%d", &id); err != nil {
		return 0, false, false
	}
	return id, deleted, true
}
