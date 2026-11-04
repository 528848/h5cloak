package replay

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Observed 网关在 X-Cloak-Debug: decide 下返回的判定。
type Observed struct {
	Status           int      `json:"status"`
	Content          string   `json:"content"`
	TouchType        string   `json:"touchType"`
	Channel          string   `json:"channel"`
	Code             string   `json:"code"`
	Attributed       bool     `json:"attributed"`
	PathRule         string   `json:"pathRule,omitempty"`
	Trace            []Step   `json:"trace,omitempty"`
	ValidationErrors []string `json:"validationErrors,omitempty"`
	Error            string   `json:"error,omitempty"`
}

type Step struct {
	Step   string `json:"step"`
	Result string `json:"result"`
}

// Result 一条用例的回放结果。
type Result struct {
	Case     Case          `json:"case"`
	Observed Observed      `json:"observed"`
	Class    string        `json:"class"`
	Reason   string        `json:"reason"`
	Latency  time.Duration `json:"latency"`
}

// Options 回放参数。
type Options struct {
	Target      string // 网关地址，如 http://127.0.0.1:8090
	Concurrency int
	Rate        int // 每秒请求数，0 不限
	Loop        int // 重复次数，压测用
	Timeout     time.Duration
	// Record 为 true 时网关照常计额度、写决策日志；默认干跑（带 X-Cloak-Replay 头）
	Record bool
	// 每条结果的回调，可为 nil
	OnResult func(Result)
}

// Runner 回放执行器。
type Runner struct {
	opt    Options
	client *http.Client
}

func NewRunner(opt Options) *Runner {
	if opt.Concurrency <= 0 {
		opt.Concurrency = 4
	}
	if opt.Loop <= 0 {
		opt.Loop = 1
	}
	if opt.Timeout <= 0 {
		opt.Timeout = 10 * time.Second
	}
	opt.Target = strings.TrimRight(opt.Target, "/")
	tr := &http.Transport{MaxIdleConnsPerHost: opt.Concurrency * 2, DisableCompression: true}
	return &Runner{opt: opt, client: &http.Client{Transport: tr, Timeout: opt.Timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

// Run 按并发与速率回放全部用例，返回结果（顺序不保证）。
func (r *Runner) Run(ctx context.Context, cases []Case) []Result {
	total := len(cases) * r.opt.Loop
	jobs := make(chan Case)
	results := make([]Result, 0, total)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < r.opt.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for c := range jobs {
				res := r.one(ctx, c)
				mu.Lock()
				results = append(results, res)
				mu.Unlock()
				if r.opt.OnResult != nil {
					r.opt.OnResult(res)
				}
			}
		}()
	}
	var tick <-chan time.Time
	if r.opt.Rate > 0 {
		t := time.NewTicker(time.Second / time.Duration(r.opt.Rate))
		defer t.Stop()
		tick = t.C
	}
send:
	for l := 0; l < r.opt.Loop; l++ {
		for _, c := range cases {
			if tick != nil {
				select {
				case <-ctx.Done():
					break send
				case <-tick:
				}
			}
			select {
			case <-ctx.Done():
				break send
			case jobs <- c:
			}
		}
	}
	close(jobs)
	wg.Wait()
	return results
}

func (r *Runner) one(ctx context.Context, c Case) Result {
	res := Result{Case: c}
	req, err := http.NewRequestWithContext(ctx, c.Method, r.opt.Target+c.URL(), nil)
	if err != nil {
		res.Observed.Error = err.Error()
		res.Class, res.Reason = ClassTransport, err.Error()
		return res
	}
	req.Host = c.Host
	req.Header.Set("X-Cloak-Debug", "decide")
	if !r.opt.Record {
		req.Header.Set("X-Cloak-Replay", "1")
	}
	req.Header.Set("Accept", "application/json")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	if c.Referer != "" {
		req.Header.Set("Referer", c.Referer)
	}
	if c.IP != "" {
		req.Header.Set("X-Forwarded-For", c.IP)
	}
	start := time.Now()
	resp, err := r.client.Do(req)
	res.Latency = time.Since(start)
	if err != nil {
		res.Observed.Error = err.Error()
		res.Class, res.Reason = ClassTransport, err.Error()
		return res
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	res.Observed.Status = resp.StatusCode
	if resp.Header.Get("X-Cloak-Content") == "" && resp.StatusCode == http.StatusNotFound {
		res.Class, res.Reason = ClassHostUnbound, "网关不认识该 Host"
		return res
	}
	var raw struct {
		Content          string           `json:"content"`
		TouchType        string           `json:"touchType"`
		Channel          string           `json:"channel"`
		Code             json.RawMessage  `json:"code"` // 字符串或 {code:...} 对象
		Attributed       bool             `json:"attributed"`
		PathRule         json.RawMessage  `json:"pathRule"`
		Trace            []Step           `json:"trace"`
		ValidationErrors []map[string]any `json:"validationErrors"`
	}
	if err := json.Unmarshal(body, &raw); err != nil || raw.Content == "" {
		res.Observed.Error = "非判定 JSON 响应"
		res.Class, res.Reason = ClassTransport, res.Observed.Error+" (HTTP "+itoa(resp.StatusCode)+")"
		return res
	}
	res.Observed.Content, res.Observed.TouchType, res.Observed.Channel, res.Observed.Attributed =
		raw.Content, raw.TouchType, raw.Channel, raw.Attributed
	res.Observed.Code = codeString(raw.Code)
	res.Observed.PathRule = pathRuleString(raw.PathRule)
	res.Observed.Trace = raw.Trace
	for _, ve := range raw.ValidationErrors {
		if s, ok := ve["reason"].(string); ok {
			res.Observed.ValidationErrors = append(res.Observed.ValidationErrors, s)
		}
	}
	res.Class, res.Reason = Classify(c, res.Observed)
	return res
}

// Latencies 排序后的耗时，供分位数计算。
func Latencies(results []Result) []time.Duration {
	out := make([]time.Duration, 0, len(results))
	for _, r := range results {
		if r.Latency > 0 {
			out = append(out, r.Latency)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Percentile 取分位数，p 为 0-100。
func Percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p / 100)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func itoa(n int) string { return strconv.Itoa(n) }

// codeString 兼容两种形态：开放接口返回字符串，网关调试输出返回 {code:...} 对象。
func codeString(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var obj struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(raw, &obj)
	return obj.Code
}

func pathRuleString(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var obj struct {
		Action  string `json:"action"`
		Pattern string `json:"pattern"`
	}
	if json.Unmarshal(raw, &obj) == nil && obj.Action != "" {
		return obj.Action + " " + obj.Pattern
	}
	return ""
}
