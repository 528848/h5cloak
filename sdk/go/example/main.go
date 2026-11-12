// 最小示例：用一行中间件把判定 + 深度代理接到任意 net/http 服务。
//
//	go run ./example
//
// 环境变量：CLOAK_API_BASE、CLOAK_SECRET_KEY、CLOAK_REVIEW_ORIGIN。
package main

import (
	"log"
	"net/http"
	"os"

	cloak "github.com/h5cloak/cloak-sdk-go"
)

func main() {
	c := cloak.New(cloak.Config{
		APIBase:      env("CLOAK_API_BASE", "https://gw.example.com"),
		SecretKey:    env("CLOAK_SECRET_KEY", "sk_xxx"),
		ReviewOrigin: env("CLOAK_REVIEW_ORIGIN", "https://review.com"),
	})

	// 你自己的真实业务内容
	real := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<h1>真实站点</h1>"))
	})

	// 一行接入：放行走 real，审核在本机深度代理审核站
	mux := http.NewServeMux()
	mux.Handle("/", c.Handle(real))

	addr := env("ADDR", ":8080")
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
