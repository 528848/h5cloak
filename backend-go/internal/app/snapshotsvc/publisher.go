package snapshotsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"cloak/internal/adapter/mysql"
	"cloak/internal/engine/snapshot"
)

// Redis key
const (
	KeySitePrefix = "cfg:site:"
	KeyHostPrefix = "host:"
	KeySiteIndex  = "cfg:sites"
	ChanChanged   = "cfg:changed"
)

// Publisher 构建并发布快照。rdb 为 nil 时只构建不发布（本地无 Redis）。
type Publisher struct {
	builder *Builder
	rdb     *goredis.Client
	log     *zap.Logger
}

func NewPublisher(b *Builder, rdb *goredis.Client, log *zap.Logger) *Publisher {
	return &Publisher{builder: b, rdb: rdb, log: log}
}

// Publish 重建站点快照并写入 Redis，然后广播变更。
// 站点已删除时清理 key 并广播删除。调用方通常异步调用，失败只记日志。
func (p *Publisher) Publish(ctx context.Context, siteID uint64) error {
	snap, err := p.builder.Build(ctx, siteID)
	if errors.Is(err, mysql.ErrNotFound) {
		return p.Remove(ctx, siteID)
	}
	if err != nil {
		return err
	}
	if p.rdb == nil {
		return nil
	}
	raw, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	pipe := p.rdb.Pipeline()
	pipe.Set(ctx, KeySitePrefix+itoa(siteID), raw, 0)
	pipe.SAdd(ctx, KeySiteIndex, itoa(siteID))
	// 旧域名索引：先清掉指向该站点的 host 再写新的
	oldHosts, _ := p.rdb.SMembers(ctx, "cfg:site:"+itoa(siteID)+":hosts").Result()
	for _, h := range oldHosts {
		pipe.Del(ctx, KeyHostPrefix+h)
	}
	pipe.Del(ctx, "cfg:site:"+itoa(siteID)+":hosts")
	for _, d := range snap.Domains {
		if d.Status == "bound" {
			pipe.Set(ctx, KeyHostPrefix+d.Domain, itoa(siteID), 0)
			pipe.SAdd(ctx, "cfg:site:"+itoa(siteID)+":hosts", d.Domain)
		}
	}
	pipe.Publish(ctx, ChanChanged, "site:"+itoa(siteID))
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}
	p.log.Info("snapshot published", zap.Uint64("site", siteID), zap.Int64("revision", snap.Revision),
		zap.Int("codes", len(snap.Codes)), zap.Int("params", len(snap.Params)))
	return nil
}

// Remove 删除站点快照。
func (p *Publisher) Remove(ctx context.Context, siteID uint64) error {
	if p.rdb == nil {
		return nil
	}
	hosts, _ := p.rdb.SMembers(ctx, "cfg:site:"+itoa(siteID)+":hosts").Result()
	pipe := p.rdb.Pipeline()
	for _, h := range hosts {
		pipe.Del(ctx, KeyHostPrefix+h)
	}
	pipe.Del(ctx, "cfg:site:"+itoa(siteID)+":hosts", KeySitePrefix+itoa(siteID))
	pipe.SRem(ctx, KeySiteIndex, itoa(siteID))
	pipe.Publish(ctx, ChanChanged, "site:"+itoa(siteID)+":deleted")
	_, err := pipe.Exec(ctx)
	return err
}

// PublishAsync 后台发布，超时 10 秒，失败只记日志。
func (p *Publisher) PublishAsync(siteID uint64) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := p.Publish(ctx, siteID); err != nil {
			p.log.Warn("snapshot publish failed", zap.Uint64("site", siteID), zap.Error(err))
		}
	}()
}

// PublishMerchantAsync 商户状态变化时重建其全部站点。
func (p *Publisher) PublishMerchantAsync(ctx context.Context, merchantID uint64) {
	rows, _, err := p.builder.sites.ListSites(ctx, merchantID, "", "", 1, 200)
	if err != nil {
		p.log.Warn("list sites for publish", zap.Error(err))
		return
	}
	for _, s := range rows {
		p.PublishAsync(s.ID)
	}
}

// PublishAll 重建全部站点（启动或内置字典变更时）。
func (p *Publisher) PublishAll(ctx context.Context) (int, error) {
	rows, _, err := p.builder.sites.ListSites(ctx, 0, "", "", 1, 200)
	if err != nil {
		return 0, err
	}
	n := 0
	var firstErr error
	for _, s := range rows {
		if err := p.Publish(ctx, s.ID); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		n++
	}
	return n, firstErr
}

// Load 网关侧：从 Redis 读取一个站点快照。
func Load(ctx context.Context, rdb *goredis.Client, siteID uint64) (*snapshot.Site, error) {
	raw, err := rdb.Get(ctx, KeySitePrefix+itoa(siteID)).Bytes()
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

// ResolveHost 网关侧：域名 → 站点 ID。
func ResolveHost(ctx context.Context, rdb *goredis.Client, host string) (uint64, error) {
	host = strings.ToLower(host)
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	v, err := rdb.Get(ctx, KeyHostPrefix+host).Result()
	if err != nil {
		return 0, err
	}
	var id uint64
	_, err = fmt.Sscanf(v, "%d", &id)
	return id, err
}

// ParseChange 解析变更消息，返回站点 ID 与是否删除。
func ParseChange(msg string) (uint64, bool, bool) {
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

func itoa(n uint64) string { return fmt.Sprint(n) }
