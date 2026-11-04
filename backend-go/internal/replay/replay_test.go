package replay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cloak/internal/domain/decisionlog"
)

func TestFromRecordRebuildsQuery(t *testing.T) {
	r := &decisionlog.Record{
		RequestID: "r1", Host: "demo.example.com", Path: "/lp", Method: "GET",
		QueryParamNames: []byte(`["ref","fbclid","utm_source"]`), MatchedParams: []byte(`["ref","utm_source"]`),
		Channel: "distributor:ALICE", Code: "ALICE", Content: "real", TouchType: "first", Attributed: true,
		IPPrefix: "203.0.113.0/24", UASummary: "UA", RefererSummary: "https://facebook.com/",
		Trace: []byte(`[{"step":"token","result":"absent"},{"step":"global","result":"emergency_review"}]`),
	}
	c := FromRecord(r)
	if len(c.Overrides) != 1 || c.Overrides[0] != "global:emergency_review" {
		t.Fatalf("overrides from trace: %v", c.Overrides)
	}
	q, _ := url.ParseQuery(c.Query)
	if q.Get("ref") != "ALICE" {
		t.Fatalf("code param should carry the code, got %q", q.Get("ref"))
	}
	if q.Get("utm_source") != "replay" {
		t.Fatalf("utm_source has no source channel, expect placeholder, got %q", q.Get("utm_source"))
	}
	if q.Get("fbclid") != "replay" || len(c.Reconstructed) != 2 {
		t.Fatalf("unmatched params must be placeholders and recorded: %q %v", c.Query, c.Reconstructed)
	}
	if c.IP != "203.0.113.0" || c.URL() == "/lp" {
		t.Fatalf("ip/url: %s %s", c.IP, c.URL())
	}
	r2 := &decisionlog.Record{Host: "h", Path: "/", QueryParamNames: []byte(`["utm_source"]`), MatchedParams: []byte(`["utm_source"]`), Channel: "source:facebook"}
	if c2 := FromRecord(r2); c2.Query != "utm_source=facebook" || len(c2.Reconstructed) != 0 {
		t.Fatalf("source channel should restore utm_source: %q %v", c2.Query, c2.Reconstructed)
	}
}

func TestClassify(t *testing.T) {
	base := Case{Expected: Expected{Content: "real", TouchType: "first", Attributed: true, Channel: "distributor:ALICE", Code: "ALICE"}}
	okObs := Observed{Content: "real", TouchType: "first", Attributed: true, Channel: "distributor:ALICE", Code: "ALICE"}
	if cls, _ := Classify(base, okObs); cls != ClassMatch {
		t.Fatalf("match: %s", cls)
	}
	if cls, _ := Classify(base, Observed{Content: "review", TouchType: "none", Trace: []Step{{"global", "emergency_review"}}}); cls != ClassGlobal {
		t.Fatalf("global: %s", cls)
	}
	if cls, _ := Classify(base, Observed{Content: "review", Trace: []Step{{"quota", "over:review_all"}}}); cls != ClassQuota {
		t.Fatalf("quota: %s", cls)
	}
	if cls, _ := Classify(base, Observed{Content: "review", Trace: []Step{{"site", "not_serving:review"}}}); cls != ClassSiteNotServe {
		t.Fatalf("site: %s", cls)
	}
	was := base
	was.Expected.Content = "review"
	was.Overrides = []string{"global:emergency_review"}
	if cls, reason := Classify(was, okObs); cls != ClassGlobal || !strings.Contains(reason, "现已解除") {
		t.Fatalf("past override: %s %s", cls, reason)
	}
	if cls, reason := Classify(was, Observed{Content: "review", TouchType: "none", PathRule: "always_review /lp",
		Trace: []Step{{"path_rule", "always_review /lp"}}}); cls != ClassUnexplained || !strings.Contains(reason, "路径规则变化") {
		t.Fatalf("path rule change beats past override: %s %s", cls, reason)
	}
	ret := base
	ret.Expected.TouchType = "return"
	if cls, _ := Classify(ret, Observed{Content: "review", TouchType: "none"}); cls != ClassNeedsToken {
		t.Fatalf("needs token: %s", cls)
	}
	rec := base
	rec.Reconstructed = []string{"query:fbclid"}
	if cls, _ := Classify(rec, Observed{Content: "review", TouchType: "none", ValidationErrors: []string{"format"}}); cls != ClassReconstructed {
		t.Fatalf("reconstructed: %s", cls)
	}
	if cls, reason := Classify(base, Observed{Content: "review", TouchType: "none"}); cls != ClassUnexplained || !strings.Contains(reason, "real → review") {
		t.Fatalf("unexplained: %s %s", cls, reason)
	}
	if Explained(ClassUnexplained) || !Explained(ClassNeedsToken) {
		t.Fatal("explained flags")
	}
}

func TestRunnerAgainstFakeGateway(t *testing.T) {
	var gotHost, gotXFF, gotDebug string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHost, gotXFF, gotDebug = r.Host, r.Header.Get("X-Forwarded-For"), r.Header.Get("X-Cloak-Debug")
		if r.Host == "unknown.example.com" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("X-Cloak-Content", "real")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": "real", "touchType": "first", "attributed": true, "channel": "distributor:ALICE",
			"code":  map[string]any{"id": 549, "code": "ALICE", "type": "distributor"},
			"trace": []map[string]string{{"step": "params", "result": "ref"}},
		})
	}))
	defer srv.Close()

	cases := []Case{
		{ID: "a", Host: "demo.example.com", Path: "/lp", Query: "ref=ALICE", IP: "203.0.113.5",
			Expected: Expected{Content: "real", TouchType: "first", Attributed: true, Channel: "distributor:ALICE", Code: "ALICE"}},
		{ID: "b", Host: "unknown.example.com", Path: "/", Expected: Expected{Content: "real", TouchType: "first"}},
	}
	results := NewRunner(Options{Target: srv.URL, Concurrency: 2}).Run(context.Background(), cases)
	if len(results) != 2 {
		t.Fatalf("results %d", len(results))
	}
	byID := map[string]Result{}
	for _, r := range results {
		byID[r.Case.ID] = r
	}
	if byID["a"].Class != ClassMatch || byID["b"].Class != ClassHostUnbound {
		t.Fatalf("classes: %s %s", byID["a"].Class, byID["b"].Class)
	}
	if gotDebug != "decide" || (gotHost != "demo.example.com" && gotHost != "unknown.example.com") || (gotXFF != "203.0.113.5" && gotXFF != "") {
		t.Fatalf("headers host=%s xff=%s debug=%s", gotHost, gotXFF, gotDebug)
	}
	sum := Summarize(srv.URL, time.Now(), time.Second, results, 5)
	if sum.Total != 2 || sum.Unexplained != 0 || sum.ByClass[ClassHostUnbound] != 1 {
		t.Fatalf("summary %+v", sum.ByClass)
	}
	if md := sum.Markdown(); !strings.Contains(md, "host_unbound") || !strings.Contains(md, "unknown.example.com/") {
		t.Fatalf("markdown missing sample:\n%s", md)
	}
}

func TestReadWriteJSONL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.jsonl")
	if err := WriteJSONL(path, []Case{{ID: "x", Host: "h", Path: "/p", Query: "a=1"}}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cases, err := ReadJSONL(strings.NewReader("# comment\n" + string(raw) + "\n\n"))
	if err != nil || len(cases) != 1 || cases[0].URL() != "/p?a=1" || cases[0].Method != "GET" {
		t.Fatalf("read back: %v %+v", err, cases)
	}
}
