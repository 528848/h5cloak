// mockorigin 本地模拟源站：-kind real|review，返回带标记的 HTML，并回显收到的请求头，便于验证代理改写。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9101", "listen address")
	kind := flag.String("kind", "real", "real | review")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/echo" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"kind": *kind, "host": r.Host, "path": r.URL.Path, "query": r.URL.RawQuery, "headers": r.Header})
			return
		}
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "http://"+r.Host+"/landed", http.StatusFound)
			return
		}
		if r.URL.Path == "/setcookie" {
			http.SetCookie(w, &http.Cookie{Name: "origin_sess", Value: "abc", Domain: strings.Split(r.Host, ":")[0], Path: "/"})
			fmt.Fprint(w, "cookie set")
			return
		}
		if r.URL.Path == "/sse" {
			w.Header().Set("Content-Type", "text/event-stream")
			f := w.(http.Flusher)
			for i := 1; i <= 3; i++ {
				fmt.Fprintf(w, "data: %s %d\n\n", *kind, i)
				f.Flush()
				time.Sleep(200 * time.Millisecond)
			}
			return
		}
		if r.URL.Path == "/slow" {
			time.Sleep(20 * time.Second)
			return
		}
		keys := make([]string, 0, len(r.Header))
		for k := range r.Header {
			if strings.HasPrefix(k, "X-") {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		var hdr strings.Builder
		for _, k := range keys {
			fmt.Fprintf(&hdr, "<li><code>%s</code>: %s</li>", k, strings.Join(r.Header.Values(k), ", "))
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Server", "mockorigin")
		color := "#0a7"
		title := "REAL SITE"
		if *kind == "review" {
			color = "#c60"
			title = "REVIEW SITE"
		}
		fmt.Fprintf(w, `<!doctype html><html><head><meta charset="utf-8"><title>%s</title></head>
<body style="font-family:sans-serif;padding:24px">
<h1 style="color:%s">%s</h1>
<p>Served by mock origin <b>%s</b> for host <b>%s</b>, path <code>%s</code></p>
<p><a href="/second?x=1">Second page</a> · <a href="https://example.com/">External</a></p>
<form action="/buy" method="post"><button>Buy</button></form>
<h3>Request headers from gateway</h3><ul>%s</ul>
</body></html>`, title, color, title, *kind, r.Host, r.URL.Path, hdr.String())
	})
	log.Printf("mock origin [%s] listening on %s", *kind, *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
