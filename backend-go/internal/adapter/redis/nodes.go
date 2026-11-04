package redis

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// NodeInfo 网关节点心跳。
type NodeInfo struct {
	Node       string  `json:"node"`
	Version    string  `json:"version"`
	Commit     string  `json:"commit"`
	StartedAt  string  `json:"startedAt"`
	SeenAt     string  `json:"seenAt"`
	Sites      int     `json:"sites"`
	QPS        float64 `json:"qps"`
	P99Ms      float64 `json:"p99Ms"`
	OriginErrs float64 `json:"originErrorRate"`
	Requests   uint64  `json:"requests"`
}

const nodeKeyPrefix = "gw:nodes:"

// Heartbeat 写节点状态，60 秒过期。
func Heartbeat(ctx context.Context, rdb *goredis.Client, info NodeInfo) error {
	if rdb == nil {
		return nil
	}
	info.SeenAt = time.Now().Format("2006-01-02 15:04:05")
	b, _ := json.Marshal(info)
	return rdb.Set(ctx, nodeKeyPrefix+info.Node, b, 60*time.Second).Err()
}

// Nodes 列出在线节点。
func Nodes(ctx context.Context, rdb *goredis.Client) ([]NodeInfo, error) {
	if rdb == nil {
		return nil, nil
	}
	var out []NodeInfo
	iter := rdb.Scan(ctx, 0, nodeKeyPrefix+"*", 100).Iterator()
	for iter.Next(ctx) {
		raw, err := rdb.Get(ctx, iter.Val()).Bytes()
		if err != nil {
			continue
		}
		var n NodeInfo
		if json.Unmarshal(raw, &n) == nil {
			if n.Node == "" {
				n.Node = strings.TrimPrefix(iter.Val(), nodeKeyPrefix)
			}
			out = append(out, n)
		}
	}
	return out, iter.Err()
}
