// 回放回归工具：把决策日志回放到网关，输出差异报告；也可作为压测发生器。
//
//	replay export -out corpus.jsonl [-site 25] [-merchant 1] [-from "2026-11-04 00:00"] [-to ...] [-limit 10000] [-entry hosted]
//	replay run -in corpus.jsonl [-target http://127.0.0.1:8090] [-concurrency 8] [-rate 0] [-loop 1] [-report report.md] [-samples 20]
//
// export 读取 configs/admin.yaml 里的 MySQL。run 只需要网关地址；报告按扩展名写 .md 或 .json。
// run 默认干跑：请求带 X-Cloak-Replay 头，网关不计额度、不写决策日志、不计入节点统计；加 -record 恢复正常记录。
// 退出码：run 有未解释差异时为 1，便于接入 CI。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	mysqladapter "cloak/internal/adapter/mysql"
	"cloak/internal/config"
	"cloak/internal/replay"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "export":
		runExport(os.Args[2:])
	case "run":
		runReplay(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func runExport(args []string) {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	out := fs.String("out", "corpus.jsonl", "输出文件")
	site := fs.Uint64("site", 0, "站点 ID，0 为全部")
	merchant := fs.Uint64("merchant", 0, "商户 ID，0 为全部")
	entry := fs.String("entry", "hosted", "接入方式：hosted / api_v2 / api_v1 / js，空为全部")
	from := fs.String("from", "", "开始时间，如 2026-11-04 00:00")
	to := fs.String("to", "", "结束时间（不含）")
	limit := fs.Int("limit", 10000, "最多导出条数")
	_ = fs.Parse(args)

	cfg, err := config.Load("admin")
	if err != nil {
		fail(err)
	}
	db, err := mysqladapter.Open(cfg.MySQL, zap.NewNop())
	if err != nil {
		fail(err)
	}
	f := replay.ExportFilter{SiteID: *site, MerchantID: *merchant, Entry: *entry, Limit: *limit}
	if f.From, err = parseTime(*from); err != nil {
		fail(err)
	}
	if f.To, err = parseTime(*to); err != nil {
		fail(err)
	}
	cases, skipped, err := replay.Export(context.Background(), db, f)
	if err != nil {
		fail(err)
	}
	if err := replay.WriteJSONL(*out, cases); err != nil {
		fail(err)
	}
	rebuilt := 0
	for _, c := range cases {
		if len(c.Reconstructed) > 0 {
			rebuilt++
		}
	}
	fmt.Printf("exported %d cases to %s (skipped %d without host, %d with reconstructed params)\n", len(cases), *out, skipped, rebuilt)
}

func runReplay(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	in := fs.String("in", "corpus.jsonl", "用例文件")
	target := fs.String("target", "http://127.0.0.1:8090", "网关地址")
	concurrency := fs.Int("concurrency", 8, "并发数")
	rate := fs.Int("rate", 0, "每秒请求数，0 不限")
	loop := fs.Int("loop", 1, "重复次数（压测）")
	report := fs.String("report", "replay-report.md", "报告文件，.md 或 .json")
	samples := fs.Int("samples", 20, "每类差异保留的样例数")
	timeout := fs.Duration("timeout", 10*time.Second, "单请求超时")
	quiet := fs.Bool("quiet", false, "不逐条打印差异")
	record := fs.Bool("record", false, "让网关照常计额度并写决策日志（默认干跑，不污染看板）")
	_ = fs.Parse(args)

	f, err := os.Open(*in)
	if err != nil {
		fail(err)
	}
	cases, err := replay.ReadJSONL(f)
	f.Close()
	if err != nil {
		fail(err)
	}
	if len(cases) == 0 {
		fail(fmt.Errorf("用例文件为空"))
	}
	opt := replay.Options{Target: *target, Concurrency: *concurrency, Rate: *rate, Loop: *loop, Timeout: *timeout, Record: *record}
	if !*quiet {
		opt.OnResult = func(r replay.Result) {
			if r.Class == replay.ClassMatch {
				return
			}
			fmt.Printf("[%s] %s%s  %s\n", r.Class, r.Case.Host, r.Case.URL(), r.Reason)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := time.Now()
	results := replay.NewRunner(opt).Run(ctx, cases)
	sum := replay.Summarize(*target, started, time.Since(started), results, *samples)
	if err := sum.Write(*report); err != nil {
		fail(err)
	}
	fmt.Printf("\n%d cases, %.1f req/s, p50 %.1f ms, p99 %.1f ms\n", sum.Total, sum.QPS, sum.P50Ms, sum.P99Ms)
	for _, c := range []string{replay.ClassMatch, replay.ClassNeedsToken, replay.ClassGlobal, replay.ClassQuota, replay.ClassSiteNotServe, replay.ClassReconstructed, replay.ClassHostUnbound, replay.ClassTransport, replay.ClassUnexplained} {
		if n := sum.ByClass[c]; n > 0 {
			fmt.Printf("  %-20s %d\n", c, n)
		}
	}
	fmt.Printf("report: %s\n", *report)
	if sum.Unexplained > 0 {
		fmt.Printf("%d unexplained differences\n", sum.Unexplained)
		os.Exit(1)
	}
}

func parseTime(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	for _, l := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02", time.RFC3339} {
		if t, err := time.ParseInLocation(l, s, time.Local); err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("时间格式无效: %s", s)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: replay export -out corpus.jsonl [-site N] [-merchant N] [-from T] [-to T] [-limit N] [-entry hosted]")
	fmt.Fprintln(os.Stderr, "       replay run -in corpus.jsonl [-target http://127.0.0.1:8090] [-concurrency N] [-rate N] [-loop N] [-report file.md|file.json]")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
