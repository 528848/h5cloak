package proxy

import (
	"bytes"
	"html/template"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"cloak/internal/engine/snapshot"
)

// templateCache 渲染结果按站点 revision 缓存。
type templateCache struct {
	mu sync.Mutex
	m  map[uint64]cachedTemplate
}

type cachedTemplate struct {
	revision int64
	html     []byte
	err      error
}

func newTemplateCache() *templateCache { return &templateCache{m: map[uint64]cachedTemplate{}} }

func (c *templateCache) render(site *snapshot.Site) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if v, ok := c.m[site.SiteID]; ok && v.revision == site.Revision {
		return v.html, v.err
	}
	out, err := renderTemplate(site.Review.TemplateHTML, site.Review.TemplateVars)
	c.m[site.SiteID] = cachedTemplate{revision: site.Revision, html: out, err: err}
	return out, err
}

// RenderTemplate 导出：渲染审核模板（开放接口 renderHint 用）。
func RenderTemplate(src string, vars map[string]string) ([]byte, error) {
	return renderTemplate(src, vars)
}

func renderTemplate(src string, vars map[string]string) ([]byte, error) {
	if src == "" {
		src = defaultReviewHTML
	}
	tpl, err := template.New("review").Parse(src)
	if err != nil {
		return nil, err
	}
	if vars == nil {
		vars = map[string]string{}
	}
	var b bytes.Buffer
	if err := tpl.Execute(&b, vars); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// serveStatic 从静态包目录按路径返回文件。spa 为真时，文件不存在则回入口页。
func serveStatic(w http.ResponseWriter, r *http.Request, site *snapshot.Site, spa bool) bool {
	dir := site.Review.StaticDir
	if dir == "" {
		return false
	}
	clean := path.Clean("/" + r.URL.Path)
	if strings.Contains(clean, "..") {
		clean = "/"
	}
	target := filepath.Join(dir, filepath.FromSlash(clean))
	st, err := os.Stat(target)
	if err != nil {
		if !spa {
			return false
		}
		target = staticEntryPath(dir, site.Review.StaticEntry)
		if _, err := os.Stat(target); err != nil {
			return false
		}
	} else if st.IsDir() {
		index := filepath.Join(target, "index.html")
		if _, err := os.Stat(index); err == nil {
			target = index
		} else if spa {
			target = staticEntryPath(dir, site.Review.StaticEntry)
			if _, err := os.Stat(target); err != nil {
				return false
			}
		} else {
			return false
		}
	}
	f, err := os.Open(target)
	if err != nil {
		return false
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil || fi.IsDir() {
		return false
	}
	w.Header().Set("Cache-Control", "no-store")
	// ServeContent 处理 Content-Type、Range、If-Modified-Since；路径已限定在目录内
	http.ServeContent(w, r, filepath.Base(target), fi.ModTime(), f)
	return true
}

func staticEntryPath(dir, entry string) string {
	if entry == "" {
		entry = "index.html"
	}
	return filepath.Join(dir, filepath.FromSlash(entry))
}

const defaultReviewHTML = `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.siteName}}</title><style>body{margin:0;font-family:sans-serif;background:#f6f7f9;color:#1f2329}main{max-width:720px;margin:48px auto;padding:0 20px;line-height:1.7}</style></head><body><main><h1>{{.siteName}}</h1><p>{{.slogan}}</p><p>Welcome. Please contact us through the information published on this page for more details.</p></main></body></html>`

const blockHTML = `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Forbidden</title></head><body style="font-family:sans-serif;text-align:center;padding:80px 20px"><h1>403</h1><p>Access denied.</p></body></html>`

const notFoundHTML = `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Not Found</title></head><body style="font-family:sans-serif;text-align:center;padding:80px 20px"><h1>404</h1><p>The page you are looking for does not exist.</p></body></html>`

const maintenanceHTML = `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Maintenance</title></head><body style="font-family:sans-serif;text-align:center;padding:80px 20px"><h1>We'll be right back</h1><p>The site is temporarily unavailable. Please try again in a few minutes.</p></body></html>`
