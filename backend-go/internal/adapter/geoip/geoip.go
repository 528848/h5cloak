// Package geoip 用 MaxMind mmdb 查国家。文件不存在时返回空实现，不影响服务。
package geoip

import (
	"net/netip"
	"os"

	"github.com/oschwald/maxminddb-golang"
)

// Resolver 国家解析。
type Resolver interface {
	Country(ip netip.Addr) string
	Close() error
}

type noop struct{}

func (noop) Country(netip.Addr) string { return "" }
func (noop) Close() error              { return nil }

type mmdb struct{ r *maxminddb.Reader }

type record struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
}

func (m *mmdb) Country(ip netip.Addr) string {
	if !ip.IsValid() {
		return ""
	}
	var rec record
	if err := m.r.Lookup(ip.AsSlice(), &rec); err != nil {
		return ""
	}
	return rec.Country.ISOCode
}

func (m *mmdb) Close() error { return m.r.Close() }

// Open 打开 mmdb；path 为空或文件不存在返回 noop 与 ok=false。
func Open(path string) (Resolver, bool, error) {
	if path == "" {
		return noop{}, false, nil
	}
	if _, err := os.Stat(path); err != nil {
		return noop{}, false, nil
	}
	r, err := maxminddb.Open(path)
	if err != nil {
		return noop{}, false, err
	}
	return &mmdb{r: r}, true, nil
}
