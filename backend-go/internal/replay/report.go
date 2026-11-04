package replay

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// Summary 差异报告。
type Summary struct {
	Target      string           `json:"target"`
	StartedAt   time.Time        `json:"startedAt"`
	Elapsed     time.Duration    `json:"elapsed"`
	Total       int              `json:"total"`
	QPS         float64          `json:"qps"`
	P50Ms       float64          `json:"p50Ms"`
	P95Ms       float64          `json:"p95Ms"`
	P99Ms       float64          `json:"p99Ms"`
	ByClass     map[string]int   `json:"byClass"`
	Unexplained int              `json:"unexplained"`
	Samples     map[string][]Row `json:"samples"`
}

// Row 报告里的样例行。
type Row struct {
	ID       string `json:"id"`
	Request  string `json:"request"`
	Expected string `json:"expected"`
	Observed string `json:"observed"`
	Reason   string `json:"reason"`
	Trace    string `json:"trace,omitempty"`
}

// Summarize 汇总结果，每类最多保留 maxSamples 条样例。
func Summarize(target string, started time.Time, elapsed time.Duration, results []Result, maxSamples int) Summary {
	s := Summary{Target: target, StartedAt: started, Elapsed: elapsed, Total: len(results),
		ByClass: map[string]int{}, Samples: map[string][]Row{}}
	for _, r := range results {
		s.ByClass[r.Class]++
		if !Explained(r.Class) {
			s.Unexplained++
		}
		if r.Class == ClassMatch || len(s.Samples[r.Class]) >= maxSamples {
			continue
		}
		s.Samples[r.Class] = append(s.Samples[r.Class], row(r))
	}
	lat := Latencies(results)
	s.P50Ms = ms(Percentile(lat, 50))
	s.P95Ms = ms(Percentile(lat, 95))
	s.P99Ms = ms(Percentile(lat, 99))
	if elapsed > 0 {
		s.QPS = float64(len(results)) / elapsed.Seconds()
	}
	return s
}

func row(r Result) Row {
	exp := r.Case.Expected
	obs := r.Observed
	steps := make([]string, 0, len(obs.Trace))
	for _, t := range obs.Trace {
		steps = append(steps, t.Step+"="+t.Result)
	}
	observed := obs.Content + "/" + obs.TouchType + "/" + describeAttr(obs.Attributed, obs.Channel, obs.Code)
	if obs.Error != "" {
		observed = obs.Error
	}
	return Row{
		ID:       r.Case.ID,
		Request:  r.Case.Host + r.Case.URL(),
		Expected: exp.Content + "/" + exp.TouchType + "/" + describeAttr(exp.Attributed, exp.Channel, exp.Code),
		Observed: observed,
		Reason:   r.Reason,
		Trace:    strings.Join(steps, " > "),
	}
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }

// Markdown 输出可读报告。
func (s Summary) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# 回放差异报告\n\n")
	fmt.Fprintf(&b, "- 目标：%s\n- 开始：%s\n- 用例：%d，耗时 %s，吞吐 %.1f req/s\n- 延迟：P50 %.1f ms / P95 %.1f ms / P99 %.1f ms\n- 未解释差异：**%d**\n\n",
		s.Target, s.StartedAt.Format("2006-01-02 15:04:05"), s.Total, s.Elapsed.Round(time.Millisecond), s.QPS, s.P50Ms, s.P95Ms, s.P99Ms, s.Unexplained)
	fmt.Fprintf(&b, "## 分类\n\n| 类别 | 数量 | 说明 |\n| --- | --- | --- |\n")
	for _, c := range classOrder(s.ByClass) {
		fmt.Fprintf(&b, "| %s | %d | %s |\n", c, s.ByClass[c], classDesc[c])
	}
	keys := make([]string, 0, len(s.Samples))
	for k := range s.Samples {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "\n## 样例：%s\n\n| 请求 | 期望 | 实际 | 原因 | 轨迹 |\n| --- | --- | --- | --- | --- |\n", k)
		for _, r := range s.Samples[k] {
			fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s |\n", r.Request, r.Expected, r.Observed, r.Reason, r.Trace)
		}
	}
	return b.String()
}

var classDesc = map[string]string{
	ClassMatch:         "判定一致",
	ClassNeedsToken:    "原请求依赖访客/交接令牌，回放无 Cookie，预期不同",
	ClassGlobal:        "全局开关（紧急审核或接入关闭）改写",
	ClassQuota:         "商户额度策略改写",
	ClassSiteNotServe:  "站点停用或商户未启用",
	ClassReconstructed: "导出时合成了参数值，校验结果不可比",
	ClassHostUnbound:   "网关没有该域名",
	ClassTransport:     "请求失败或响应不是判定 JSON",
	ClassUnexplained:   "需要排查",
}

func classOrder(by map[string]int) []string {
	order := []string{ClassMatch, ClassNeedsToken, ClassGlobal, ClassQuota, ClassSiteNotServe, ClassReconstructed, ClassHostUnbound, ClassTransport, ClassUnexplained}
	out := make([]string, 0, len(by))
	for _, c := range order {
		if by[c] > 0 {
			out = append(out, c)
		}
	}
	return out
}

// Write 把报告写成 .md 或 .json（按扩展名）。
func (s Summary) Write(path string) error {
	if strings.HasSuffix(strings.ToLower(path), ".json") {
		b, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(path, b, 0o644)
	}
	return os.WriteFile(path, []byte(s.Markdown()), 0o644)
}
