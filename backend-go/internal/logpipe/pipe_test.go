package logpipe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestWALWithoutRedis(t *testing.T) {
	dir := t.TempDir()
	p := New(Options{BufferSize: 10, BatchSize: 3, FlushInterval: 20 * time.Millisecond, WALDir: dir}, nil, zap.NewNop())
	for i := 0; i < 7; i++ {
		p.Publish("decision_logs", map[string]int{"i": i})
	}
	p.Publish("attribution_writes", map[string]string{"a": "b"})
	p.Close()
	st := p.Stats()
	if st.Walled != 8 || st.Dropped != 0 || st.RedisOK {
		t.Fatalf("stats %+v", st)
	}
	b, err := os.ReadFile(filepath.Join(dir, "decision_logs.wal"))
	if err != nil || strings.Count(string(b), "\n") != 7 {
		t.Fatalf("wal content: %v %q", err, b)
	}
	if _, err := os.Stat(filepath.Join(dir, "attribution_writes.wal")); err != nil {
		t.Fatal("second stream wal missing")
	}
}

func TestDropWhenFullAndNoWAL(t *testing.T) {
	p := New(Options{BufferSize: 2, BatchSize: 100, FlushInterval: time.Hour}, nil, zap.NewNop())
	for i := 0; i < 5; i++ {
		p.Publish("s", i)
	}
	if p.Stats().Dropped < 3 {
		t.Fatalf("expected drops: %+v", p.Stats())
	}
	p.Close()
	// 无 WAL 目录：flush 时全部丢弃
	if p.Stats().Dropped != 5 {
		t.Fatalf("all dropped: %+v", p.Stats())
	}
}
