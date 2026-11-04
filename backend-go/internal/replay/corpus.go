// Package replay 把决策日志回放到网关并输出差异报告。
//
// 用途：版本升级前的回归对比、配置改动后的影响评估，以及压测时的真实流量发生器。
// 日志里没有参数值和访客令牌，导出时会按归因结果重建参数并标注，差异报告据此解释。
package replay

import (
	"bufio"
	"encoding/json"
	"io"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"cloak/internal/domain/decisionlog"
)

// Expected 日志中记录的判定结果。
type Expected struct {
	Content    string `json:"content"`
	TouchType  string `json:"touchType"`
	Channel    string `json:"channel,omitempty"`
	Code       string `json:"code,omitempty"`
	Attributed bool   `json:"attributed"`
	PathRule   string `json:"pathRule,omitempty"`
}

// Case 一条可回放的请求。
type Case struct {
	ID            string    `json:"id"`
	CreatedAt     time.Time `json:"createdAt"`
	SiteID        uint64    `json:"siteId"`
	Entry         string    `json:"entry"`
	Method        string    `json:"method"`
	Host          string    `json:"host"`
	Path          string    `json:"path"`
	Query         string    `json:"query,omitempty"`
	UserAgent     string    `json:"userAgent,omitempty"`
	Referer       string    `json:"referer,omitempty"`
	IP            string    `json:"ip,omitempty"`
	Expected      Expected  `json:"expected"`
	Reconstructed []string  `json:"reconstructed,omitempty"` // 导出时合成的部分，如 query:fbclid
	// Overrides 原请求当时被哪些开关改写过（global / quota / site 轨迹步骤），回放时这些条件可能已不存在
	Overrides []string `json:"overrides,omitempty"`
}

// URL 回放用的路径与查询串。
func (c *Case) URL() string {
	if c.Query == "" {
		return c.Path
	}
	return c.Path + "?" + c.Query
}

// 参数名到合成值的规则：utm_source 取渠道里的来源；其余点击 ID 一类给固定占位。
var sourceParams = map[string]bool{"utm_source": true, "source": true, "src": true}

// FromRecord 把一条决策日志转为回放用例。参数值日志里没有，这里按归因结果重建。
func FromRecord(r *decisionlog.Record) Case {
	c := Case{
		ID: r.RequestID, CreatedAt: r.CreatedAt, SiteID: r.SiteID, Entry: r.Entry,
		Method: nonEmpty(r.Method, "GET"), Host: r.Host, Path: nonEmpty(r.Path, "/"),
		UserAgent: r.UASummary, Referer: r.RefererSummary, IP: ipFromPrefix(r.IPPrefix),
		Expected: Expected{
			Content: r.Content, TouchType: r.TouchType, Channel: r.Channel, Code: r.Code,
			Attributed: r.Attributed, PathRule: r.PathRule,
		},
	}
	var names, matched []string
	_ = json.Unmarshal(r.QueryParamNames, &names)
	_ = json.Unmarshal(r.MatchedParams, &matched)
	c.Query, c.Reconstructed = rebuildQuery(names, matched, r.Channel, r.Code)
	var trace []Step
	_ = json.Unmarshal(r.Trace, &trace)
	for _, s := range trace {
		if s.Step == "global" || s.Step == "quota" || (s.Step == "site" && strings.HasPrefix(s.Result, "not_serving")) {
			c.Overrides = append(c.Overrides, s.Step+":"+s.Result)
		}
	}
	return c
}

// rebuildQuery 用参数名列表重建查询串。
//   - 命中的参数里，渠道为分销码/活动码时把码值给第一个非来源参数；
//   - utm_source 一类从 channel 的 source:xxx 取值；
//   - 其余参数给占位值 replay，并记录在 reconstructed 里。
func rebuildQuery(names, matched []string, channel, code string) (string, []string) {
	if len(names) == 0 {
		return "", nil
	}
	sort.Strings(names)
	isMatched := map[string]bool{}
	for _, m := range matched {
		isMatched[m] = true
	}
	source := ""
	if strings.HasPrefix(channel, "source:") {
		source = strings.TrimPrefix(channel, "source:")
	}
	codeChannel := strings.HasPrefix(channel, "distributor:") || strings.HasPrefix(channel, "campaign:")
	codeAssigned := false
	values := url.Values{}
	var rebuilt []string
	for _, n := range names {
		switch {
		case sourceParams[strings.ToLower(n)] && source != "":
			values.Set(n, source)
		case isMatched[n] && codeChannel && code != "" && !codeAssigned && !sourceParams[strings.ToLower(n)]:
			values.Set(n, code)
			codeAssigned = true
		default:
			values.Set(n, "replay")
			rebuilt = append(rebuilt, "query:"+n)
		}
	}
	return values.Encode(), rebuilt
}

func ipFromPrefix(prefix string) string {
	if prefix == "" {
		return ""
	}
	if i := strings.IndexByte(prefix, '/'); i >= 0 {
		prefix = prefix[:i]
	}
	return prefix
}

func nonEmpty(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// WriteJSONL 写用例文件。
func WriteJSONL(path string, cases []Case) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)
	for i := range cases {
		if err := enc.Encode(&cases[i]); err != nil {
			return err
		}
	}
	return w.Flush()
}

// ReadJSONL 读用例文件，空行与 # 开头的行跳过。
func ReadJSONL(r io.Reader) ([]Case, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), 4<<20)
	var out []Case
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var c Case
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			return nil, err
		}
		if c.Method == "" {
			c.Method = "GET"
		}
		if c.Path == "" {
			c.Path = "/"
		}
		out = append(out, c)
	}
	return out, sc.Err()
}
