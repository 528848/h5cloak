// Package logpipe 网关侧日志管道：内存环形缓冲 → 批量 XADD 到 Redis Stream；
// Redis 不可用时写本地 WAL，恢复后回放。写入永不阻塞请求。
package logpipe

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Options 管道参数。
type Options struct {
	BufferSize    int           // 环形缓冲容量，默认 10000
	BatchSize     int           // 每批最多条数，默认 500
	FlushInterval time.Duration // 刷新间隔，默认 200ms
	StreamMaxLen  int64         // XADD MAXLEN ~，默认 1,000,000
	WALDir        string        // 本地 WAL 目录，空则不落盘
	WALMaxBytes   int64         // WAL 单流上限，默认 256MB，超过丢最旧
}

type item struct {
	stream  string
	payload []byte
}

// Pipe 日志管道。
type Pipe struct {
	opt     Options
	rdb     *goredis.Client
	log     *zap.Logger
	ch      chan item
	dropped atomic.Int64
	sent    atomic.Int64
	walled  atomic.Int64
	wg      sync.WaitGroup
	stop    chan struct{}
	walMu   sync.Mutex
	redisOK atomic.Bool
}

// New 创建并启动管道。rdb 可为 nil（只写 WAL）。
func New(opt Options, rdb *goredis.Client, log *zap.Logger) *Pipe {
	if opt.BufferSize <= 0 {
		opt.BufferSize = 10000
	}
	if opt.BatchSize <= 0 {
		opt.BatchSize = 500
	}
	if opt.FlushInterval <= 0 {
		opt.FlushInterval = 200 * time.Millisecond
	}
	if opt.StreamMaxLen <= 0 {
		opt.StreamMaxLen = 1_000_000
	}
	if opt.WALMaxBytes <= 0 {
		opt.WALMaxBytes = 256 << 20
	}
	p := &Pipe{opt: opt, rdb: rdb, log: log, ch: make(chan item, opt.BufferSize), stop: make(chan struct{})}
	p.redisOK.Store(rdb != nil)
	if opt.WALDir != "" {
		_ = os.MkdirAll(opt.WALDir, 0o755)
	}
	p.wg.Add(1)
	go p.loop()
	if rdb != nil && opt.WALDir != "" {
		p.wg.Add(1)
		go p.replayLoop()
	}
	return p
}

// Publish 把事件放入缓冲；满了直接丢弃并计数，不阻塞调用方。
func (p *Pipe) Publish(stream string, v any) {
	raw, err := json.Marshal(v)
	if err != nil {
		return
	}
	select {
	case p.ch <- item{stream: stream, payload: raw}:
	default:
		p.dropped.Add(1)
	}
}

// Close 刷完缓冲后退出。
func (p *Pipe) Close() {
	close(p.stop)
	p.wg.Wait()
}

// Stats 运行统计。
type Stats struct {
	Buffered int   `json:"buffered"`
	Sent     int64 `json:"sent"`
	Dropped  int64 `json:"dropped"`
	Walled   int64 `json:"walled"`
	RedisOK  bool  `json:"redisOk"`
}

func (p *Pipe) Stats() Stats {
	return Stats{Buffered: len(p.ch), Sent: p.sent.Load(), Dropped: p.dropped.Load(), Walled: p.walled.Load(), RedisOK: p.redisOK.Load()}
}

func (p *Pipe) loop() {
	defer p.wg.Done()
	t := time.NewTicker(p.opt.FlushInterval)
	defer t.Stop()
	batch := make([]item, 0, p.opt.BatchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		p.flush(batch)
		batch = batch[:0]
	}
	for {
		select {
		case it := <-p.ch:
			batch = append(batch, it)
			if len(batch) >= p.opt.BatchSize {
				flush()
			}
		case <-t.C:
			flush()
		case <-p.stop:
			// 排空
			for {
				select {
				case it := <-p.ch:
					batch = append(batch, it)
					if len(batch) >= p.opt.BatchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}
		}
	}
}

func (p *Pipe) flush(batch []item) {
	if p.rdb == nil || !p.redisOK.Load() {
		p.toWAL(batch)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	pipe := p.rdb.Pipeline()
	for _, it := range batch {
		pipe.XAdd(ctx, &goredis.XAddArgs{Stream: it.stream, MaxLen: p.opt.StreamMaxLen, Approx: true, Values: map[string]any{"d": it.payload}})
	}
	if _, err := pipe.Exec(ctx); err != nil {
		p.redisOK.Store(false)
		p.log.Warn("logpipe redis write failed, switching to WAL", zap.Error(err), zap.Int("batch", len(batch)))
		p.toWAL(batch)
		return
	}
	p.sent.Add(int64(len(batch)))
}

// ---- WAL ----

func (p *Pipe) walPath(stream string) string {
	return filepath.Join(p.opt.WALDir, stream+".wal")
}

func (p *Pipe) toWAL(batch []item) {
	if p.opt.WALDir == "" {
		p.dropped.Add(int64(len(batch)))
		return
	}
	p.walMu.Lock()
	defer p.walMu.Unlock()
	files := map[string]*os.File{}
	defer func() {
		for _, f := range files {
			_ = f.Close()
		}
	}()
	for _, it := range batch {
		f, ok := files[it.stream]
		if !ok {
			var err error
			f, err = os.OpenFile(p.walPath(it.stream), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
			if err != nil {
				p.dropped.Add(1)
				continue
			}
			files[it.stream] = f
			if st, err := f.Stat(); err == nil && st.Size() > p.opt.WALMaxBytes {
				// 超容量：截断旧内容（丢最旧）
				_ = f.Truncate(0)
				_, _ = f.Seek(0, 0)
			}
		}
		_, _ = f.Write(append(it.payload, '\n'))
		p.walled.Add(1)
	}
}

// replayLoop 定期探测 Redis，恢复后把 WAL 回放到 Stream。
func (p *Pipe) replayLoop() {
	defer p.wg.Done()
	t := time.NewTicker(3 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-p.stop:
			return
		case <-t.C:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			err := p.rdb.Ping(ctx).Err()
			cancel()
			if err != nil {
				p.redisOK.Store(false)
				continue
			}
			if !p.redisOK.Load() {
				p.log.Info("logpipe redis recovered")
				p.redisOK.Store(true)
			}
			p.replay()
		}
	}
}

// replay 把 WAL 文件内容批量写入 Redis，成功后删除文件。
func (p *Pipe) replay() {
	p.walMu.Lock()
	defer p.walMu.Unlock()
	entries, err := os.ReadDir(p.opt.WALDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".wal" {
			continue
		}
		stream := e.Name()[:len(e.Name())-4]
		path := filepath.Join(p.opt.WALDir, e.Name())
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 4<<20)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		pipe := p.rdb.Pipeline()
		n := 0
		failed := false
		for sc.Scan() {
			line := append([]byte(nil), sc.Bytes()...)
			if len(line) == 0 {
				continue
			}
			pipe.XAdd(ctx, &goredis.XAddArgs{Stream: stream, MaxLen: p.opt.StreamMaxLen, Approx: true, Values: map[string]any{"d": line}})
			n++
			if n%500 == 0 {
				if _, err := pipe.Exec(ctx); err != nil {
					failed = true
					break
				}
			}
		}
		if !failed && n%500 != 0 {
			if _, err := pipe.Exec(ctx); err != nil {
				failed = true
			}
		}
		cancel()
		_ = f.Close()
		if failed {
			p.redisOK.Store(false)
			p.log.Warn("logpipe replay failed, will retry", zap.String("stream", stream))
			return
		}
		_ = os.Remove(path)
		if n > 0 {
			p.sent.Add(int64(n))
			p.log.Info("logpipe replayed WAL", zap.String("stream", stream), zap.Int("events", n))
		}
	}
}
