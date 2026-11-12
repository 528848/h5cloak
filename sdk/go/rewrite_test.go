package cloak

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type goldenFile struct {
	Cases []goldenCase `json:"cases"`
}

type goldenCase struct {
	Name        string   `json:"name"`
	OriginHosts []string `json:"originHosts"`
	PublicHost  string   `json:"publicHost"`
	Scheme      string   `json:"scheme"`
	Input       string   `json:"input"`
	Expect      string   `json:"expect"`
}

// TestRewriteBodyHostGolden 跑共享的跨语言黄金用例集，其它语言 SDK 必须对同一文件产出一致结果。
func TestRewriteBodyHostGolden(t *testing.T) {
	path := filepath.Join("..", "testdata", "rewrite-golden.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var gf goldenFile
	if err := json.Unmarshal(raw, &gf); err != nil {
		t.Fatalf("parse golden: %v", err)
	}
	if len(gf.Cases) == 0 {
		t.Fatal("no golden cases")
	}
	for _, c := range gf.Cases {
		t.Run(c.Name, func(t *testing.T) {
			got := string(RewriteBodyHost([]byte(c.Input), c.OriginHosts, c.PublicHost, c.Scheme))
			if got != c.Expect {
				t.Errorf("case %s\n input:  %q\n expect: %q\n got:    %q", c.Name, c.Input, c.Expect, got)
			}
		})
	}
}

func TestRewritableContentType(t *testing.T) {
	yes := []string{"text/html", "text/html; charset=utf-8", "application/json", "application/ld+json", "image/svg+xml", "application/vnd.api+json"}
	no := []string{"image/png", "application/octet-stream", "font/woff2", "video/mp4", ""}
	for _, ct := range yes {
		if !RewritableContentType(ct) {
			t.Errorf("want rewritable: %q", ct)
		}
	}
	for _, ct := range no {
		if RewritableContentType(ct) {
			t.Errorf("want NOT rewritable: %q", ct)
		}
	}
}

func TestLooksLikeHTML(t *testing.T) {
	yes := [][]byte{[]byte("<!DOCTYPE html><html></html>"), []byte("  \n<html>hi</html>"), []byte("<rss>...</rss>")}
	no := [][]byte{[]byte("PNG\x00\x00binary"), {}, []byte("plain text no tag")}
	for _, b := range yes {
		if !LooksLikeHTML(b) {
			t.Errorf("want html-like: %q", b)
		}
	}
	for _, b := range no {
		if LooksLikeHTML(b) {
			t.Errorf("want NOT html-like: %q", b)
		}
	}
}
