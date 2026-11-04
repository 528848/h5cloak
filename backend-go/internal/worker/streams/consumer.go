// Package streams 消费 Redis Stream：决策日志批写、归因写入、API 日志。
package streams

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"cloak/internal/adapter/mysql"
	"cloak/internal/domain/decisionlog"
)

// Consumer 消费者。
type Consumer struct {
	rdb   *goredis.Client
	repo  *mysql.LogRepo
	log   *zap.Logger
	name  string
	stats Stats
}

// Stats 消费统计。
type Stats struct {
	Decisions    int64
	Attributions int64
	APILogs      int64
	Errors       int64
}

func New(rdb *goredis.Client, repo *mysql.LogRepo, log *zap.Logger, consumerName string) *Consumer {
	return &Consumer{rdb: rdb, repo: repo, log: log, name: consumerName}
}

// Run 启动三个流的消费循环，阻塞到 ctx 结束。
func (c *Consumer) Run(ctx context.Context) {
	for _, s := range []string{decisionlog.StreamDecisions, decisionlog.StreamAttributions, decisionlog.StreamAPILogs} {
		c.ensureGroup(ctx, s)
	}
	go c.loop(ctx, decisionlog.StreamDecisions, 500, c.handleDecisions)
	go c.loop(ctx, decisionlog.StreamAttributions, 200, c.handleAttributions)
	go c.loop(ctx, decisionlog.StreamAPILogs, 500, c.handleAPILogs)
	<-ctx.Done()
}

func (c *Consumer) ensureGroup(ctx context.Context, stream string) {
	err := c.rdb.XGroupCreateMkStream(ctx, stream, decisionlog.ConsumerGroup, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		c.log.Warn("create consumer group", zap.String("stream", stream), zap.Error(err))
	}
}

type handler func(ctx context.Context, msgs []goredis.XMessage) error

func (c *Consumer) loop(ctx context.Context, stream string, count int64, h handler) {
	// 先处理本消费者未确认的积压（上次崩溃遗留）
	c.drainPending(ctx, stream, count, h)
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		res, err := c.rdb.XReadGroup(ctx, &goredis.XReadGroupArgs{
			Group: decisionlog.ConsumerGroup, Consumer: c.name,
			Streams: []string{stream, ">"}, Count: count, Block: 2 * time.Second,
		}).Result()
		if err != nil {
			if errors.Is(err, goredis.Nil) || ctx.Err() != nil {
				continue
			}
			c.stats.Errors++
			c.log.Warn("xreadgroup", zap.String("stream", stream), zap.Error(err))
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			if backoff < 10*time.Second {
				backoff *= 2
			}
			// 连接恢复后组可能不存在（Redis 重建）
			c.ensureGroup(ctx, stream)
			continue
		}
		backoff = time.Second
		for _, st := range res {
			if len(st.Messages) == 0 {
				continue
			}
			c.process(ctx, stream, st.Messages, h)
		}
	}
}

func (c *Consumer) drainPending(ctx context.Context, stream string, count int64, h handler) {
	for {
		res, err := c.rdb.XReadGroup(ctx, &goredis.XReadGroupArgs{
			Group: decisionlog.ConsumerGroup, Consumer: c.name,
			Streams: []string{stream, "0"}, Count: count,
		}).Result()
		if err != nil || len(res) == 0 || len(res[0].Messages) == 0 {
			return
		}
		c.process(ctx, stream, res[0].Messages, h)
	}
}

func (c *Consumer) process(ctx context.Context, stream string, msgs []goredis.XMessage, h handler) {
	if err := h(ctx, msgs); err != nil {
		c.stats.Errors++
		c.log.Warn("handle batch failed, will retry as pending", zap.String("stream", stream), zap.Int("n", len(msgs)), zap.Error(err))
		return // 不 ACK，下次 drainPending 重试
	}
	ids := make([]string, 0, len(msgs))
	for _, m := range msgs {
		ids = append(ids, m.ID)
	}
	if err := c.rdb.XAck(ctx, stream, decisionlog.ConsumerGroup, ids...).Err(); err != nil {
		c.log.Warn("xack", zap.Error(err))
	}
}

func payload(m goredis.XMessage) []byte {
	switch v := m.Values["d"].(type) {
	case string:
		return []byte(v)
	case []byte:
		return v
	}
	return nil
}

func (c *Consumer) handleDecisions(ctx context.Context, msgs []goredis.XMessage) error {
	events := make([]decisionlog.Event, 0, len(msgs))
	usage := map[uint64]uint64{}
	for _, m := range msgs {
		var e decisionlog.Event
		if err := json.Unmarshal(payload(m), &e); err != nil || e.RequestID == "" {
			continue
		}
		events = append(events, e)
		if e.Content != "bypass" {
			usage[e.MerchantID]++
		}
	}
	n, err := c.repo.InsertDecisions(ctx, events)
	if err != nil {
		return err
	}
	c.stats.Decisions += n
	// 月度计量按事件时间归月
	for mid, delta := range usage {
		period := time.Now().Format("200601")
		if len(events) > 0 {
			period = events[0].CreatedAt.Format("200601")
		}
		if err := c.repo.AddUsage(ctx, mid, decisionlog.MetricDecisions, period, delta); err != nil {
			c.log.Warn("add usage", zap.Error(err))
		}
	}
	return nil
}

func (c *Consumer) handleAttributions(ctx context.Context, msgs []goredis.XMessage) error {
	for _, m := range msgs {
		var w decisionlog.AttributionWrite
		if err := json.Unmarshal(payload(m), &w); err != nil || w.VisitorID == "" {
			continue
		}
		if err := c.repo.ApplyAttribution(ctx, &w); err != nil {
			return err
		}
		c.stats.Attributions++
		c.cacheAttribution(ctx, &w)
	}
	return nil
}

// cacheAttribution 回填 attr:{site}:{vid} 摘要，供 resolve 接口与详情页快速读取。
func (c *Consumer) cacheAttribution(ctx context.Context, w *decisionlog.AttributionWrite) {
	key := "attr:" + itoa(w.SiteID) + ":" + w.VisitorID
	ttl := 30 * 24 * time.Hour
	if w.ExpiresAt != nil {
		if d := time.Until(*w.ExpiresAt); d > 0 {
			ttl = d
		}
	}
	pipe := c.rdb.Pipeline()
	pipe.HSet(ctx, key, "aid", w.AttributionID, "ch", w.Channel, "cid", w.CodeID, "touch", w.TouchType, "at", w.At.Format(time.RFC3339))
	pipe.Expire(ctx, key, ttl)
	_, _ = pipe.Exec(ctx)
}

func (c *Consumer) handleAPILogs(ctx context.Context, msgs []goredis.XMessage) error {
	events := make([]decisionlog.APILogEvent, 0, len(msgs))
	for _, m := range msgs {
		var e decisionlog.APILogEvent
		if err := json.Unmarshal(payload(m), &e); err != nil || e.RequestID == "" {
			continue
		}
		events = append(events, e)
	}
	if err := c.repo.InsertAPILogs(ctx, events); err != nil {
		return err
	}
	c.stats.APILogs += int64(len(events))
	return nil
}

// Stats 当前统计。
func (c *Consumer) Stats() Stats { return c.stats }

func itoa(n uint64) string {
	var b [20]byte
	i := len(b)
	if n == 0 {
		return "0"
	}
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
