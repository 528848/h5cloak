package platformsvc

import (
	"archive/zip"
	"bytes"
	"context"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"cloak/internal/adapter/mysql"
	"cloak/internal/domain/identity"
	domainsite "cloak/internal/domain/site"
	"cloak/internal/pkg/probe"
)

type TemplateInput struct {
	Code              string `json:"code"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	Kind              string `json:"kind"`
	Status            string `json:"status"`
	HTML              string `json:"html"`
	URL               string `json:"url"`
	HostHeader        string `json:"hostHeader"`
	TimeoutMs         int    `json:"timeoutMs"`
	Entry             string `json:"entry"`
	SpaFallback       bool   `json:"spaFallback"`
	ConfirmKindChange bool   `json:"confirmKindChange"`
}

func (s *Service) TemplateStats(ctx context.Context) (map[string]any, error) {
	byKind, err := s.sites.CountTemplatesByKind(ctx)
	if err != nil {
		return nil, err
	}
	var total int64
	for _, n := range byKind {
		total += n
	}
	return map[string]any{"total": total, "html": byKind["html"], "url": byKind["url"], "static": byKind["static"]}, nil
}

func (s *Service) ListTemplates(ctx context.Context, keyword, kind, status string, page, size int) ([]map[string]any, int64, error) {
	rows, total, err := s.sites.ListTemplatesPage(ctx, mysql.TemplateFilter{
		Keyword: strings.TrimSpace(keyword), Kind: kind, Status: status, Page: page, Size: size,
	})
	if err != nil {
		return nil, 0, err
	}
	out := make([]map[string]any, 0, len(rows))
	for i := range rows {
		n, _ := s.sites.TemplateUseCount(ctx, rows[i].ID)
		out = append(out, templateRow(&rows[i], false, n, nil))
	}
	return out, total, nil
}

func (s *Service) GetTemplate(ctx context.Context, id uint64) (map[string]any, error) {
	t, err := s.sites.FindTemplate(ctx, id)
	if err != nil {
		return nil, errf(404, "模板不存在")
	}
	n, err := s.sites.TemplateUseCount(ctx, id)
	if err != nil {
		return nil, err
	}
	files, _ := s.sites.ListTemplateFiles(ctx, id)
	return templateRow(t, true, n, files), nil
}

func (s *Service) SaveTemplate(ctx context.Context, id uint64, in TemplateInput) (uint64, string, error) {
	in.Code = strings.TrimSpace(in.Code)
	in.Name = strings.TrimSpace(in.Name)
	in.Kind = strings.TrimSpace(in.Kind)
	if in.Kind == "" {
		in.Kind = "html"
	}
	if in.Kind != "html" && in.Kind != "url" && in.Kind != "static" {
		return 0, "", errf(400, "来源类型无效")
	}
	if in.Status == "" {
		in.Status = "enabled"
	}
	if in.Status != "enabled" && in.Status != "disabled" {
		return 0, "", errf(400, "状态无效")
	}
	if in.Name == "" || (id == 0 && in.Code == "") {
		return 0, "", errf(400, "请填写模板编码和名称")
	}
	if in.Entry == "" {
		in.Entry = "index.html"
	}
	if in.TimeoutMs <= 0 {
		in.TimeoutMs = 8000
	}
	if in.TimeoutMs > 30000 {
		in.TimeoutMs = 30000
	}
	var current *domainsite.Template
	if id > 0 {
		t, err := s.sites.FindTemplate(ctx, id)
		if err != nil {
			return 0, "", errf(404, "模板不存在")
		}
		current = t
		in.Code = t.Code
		if t.Kind != "" && t.Kind != in.Kind {
			n, err := s.sites.TemplateUseCount(ctx, id)
			if err != nil {
				return 0, "", err
			}
			if n > 0 && !in.ConfirmKindChange {
				return 0, "", errf(409, "已有站点使用该模板，确认后才会修改来源")
			}
		}
	}
	rows, err := s.sites.ListTemplates(ctx)
	if err != nil {
		return 0, "", err
	}
	for i := range rows {
		if rows[i].Code == in.Code && rows[i].ID != id {
			return 0, "", errf(400, "模板编码已存在")
		}
	}
	probeMsg := ""
	switch in.Kind {
	case "html":
		if len(in.HTML) == 0 || len(in.HTML) > 1<<20 {
			return 0, "", errf(400, "HTML 不能为空，且不超过 1MB")
		}
		if _, err := template.New("review").Parse(in.HTML); err != nil {
			return 0, "", errf(400, "HTML 模板语法错误："+err.Error())
		}
		in.URL, in.HostHeader = "", ""
	case "url":
		if err := validatePublicURL(in.URL); err != nil {
			return 0, "", err
		}
		in.HTML = ""
		if msg, err := probeURL(in.URL, in.HostHeader, time.Duration(in.TimeoutMs)*time.Millisecond); err != nil {
			probeMsg = "已保存，探测失败：" + err.Error()
		} else {
			probeMsg = msg
		}
	default:
		in.HTML, in.URL, in.HostHeader = "", "", ""
	}
	t := &domainsite.Template{
		ID: id, Code: in.Code, Name: in.Name, Description: identity.Ptr(in.Description),
		Kind: in.Kind, Status: in.Status, HTML: in.HTML, URL: strings.TrimSpace(in.URL),
		HostHeader: strings.TrimSpace(in.HostHeader), TimeoutMs: in.TimeoutMs, Entry: cleanEntry(in.Entry),
		SpaFallback: in.SpaFallback,
	}
	if current != nil {
		t.ActiveVersion = current.ActiveVersion
		if in.Kind != "static" {
			t.ActiveVersion = 0
		}
	}
	if err := s.sites.SaveTemplate(ctx, t); err != nil {
		return 0, "", err
	}
	s.republishTemplate(ctx, t.ID)
	if probeMsg == "" {
		probeMsg = "保存成功"
	}
	return t.ID, probeMsg, nil
}

func (s *Service) SetTemplateStatus(ctx context.Context, id uint64, status string) error {
	if status != "enabled" && status != "disabled" {
		return errf(400, "状态无效")
	}
	t, err := s.sites.FindTemplate(ctx, id)
	if err != nil {
		return errf(404, "模板不存在")
	}
	t.Status = status
	if err := s.sites.SaveTemplate(ctx, t); err != nil {
		return err
	}
	s.republishTemplate(ctx, id)
	return nil
}

func (s *Service) DeleteTemplate(ctx context.Context, id uint64) error {
	t, err := s.sites.FindTemplate(ctx, id)
	if err != nil {
		return errf(404, "模板不存在")
	}
	n, err := s.sites.TemplateUseCount(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return errf(409, "模板正被站点使用，请先解除引用")
	}
	files, _ := s.sites.ListTemplateFiles(ctx, id)
	if err := s.sites.DeleteTemplate(ctx, id); err != nil {
		return err
	}
	for _, f := range files {
		_ = os.RemoveAll(f.Dir)
	}
	_ = os.RemoveAll(filepath.Join(s.dataDir, "templates", t.Code))
	return nil
}

func (s *Service) UploadTemplatePackage(ctx context.Context, id uint64, r io.ReaderAt, size int64, entry string, spa bool) (map[string]any, error) {
	t, err := s.sites.FindTemplate(ctx, id)
	if err != nil {
		return nil, errf(404, "模板不存在")
	}
	if t.Kind != "static" {
		return nil, errf(400, "只有静态项目可以上传压缩包")
	}
	if size <= 0 || size > 50<<20 {
		return nil, errf(400, "压缩包须小于 50MB")
	}
	ver, err := s.sites.NextTemplateVersion(ctx, id)
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(s.dataDir, "templates", itoa(id), "v"+itoa(uint64(ver)))
	entry, files, bytesTotal, err := unzipTemplate(r, size, dir, entry)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	row := &domainsite.TemplateFile{
		TemplateID: id, Version: ver, Dir: dir, Entry: entry, Bytes: bytesTotal, FileCount: files, Status: "ready",
	}
	if err := s.sites.CreateTemplateFile(ctx, row); err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	t.ActiveVersion = ver
	t.Entry = entry
	t.SpaFallback = spa
	if err := s.sites.SaveTemplate(ctx, t); err != nil {
		return nil, err
	}
	s.pruneTemplateFiles(ctx, id)
	s.republishTemplate(ctx, id)
	return map[string]any{"version": ver, "entry": entry, "bytes": bytesTotal, "fileCount": files}, nil
}

func (s *Service) RollbackTemplate(ctx context.Context, id uint64, version int) error {
	t, err := s.sites.FindTemplate(ctx, id)
	if err != nil {
		return errf(404, "模板不存在")
	}
	f, err := s.sites.FindTemplateFile(ctx, id, version)
	if err != nil {
		return errf(400, "该版本不存在")
	}
	t.ActiveVersion = version
	t.Entry = f.Entry
	if err := s.sites.SaveTemplate(ctx, t); err != nil {
		return err
	}
	s.republishTemplate(ctx, id)
	return nil
}

func (s *Service) PreviewTemplate(ctx context.Context, id uint64, vars map[string]string) (map[string]any, error) {
	t, err := s.sites.FindTemplate(ctx, id)
	if err != nil {
		return nil, errf(404, "模板不存在")
	}
	switch t.Kind {
	case "url":
		msg, err := probeURL(t.URL, t.HostHeader, time.Duration(t.TimeoutMs)*time.Millisecond)
		if err != nil {
			return map[string]any{"ok": false, "message": err.Error()}, nil
		}
		return map[string]any{"ok": true, "message": msg}, nil
	case "static":
		return map[string]any{"ok": t.ActiveVersion > 0, "message": "静态项目请通过版本列表查看文件"}, nil
	default:
		if vars == nil {
			vars = map[string]string{}
		}
		tpl, err := template.New("review").Parse(t.HTML)
		if err != nil {
			return nil, errf(400, "HTML 模板语法错误："+err.Error())
		}
		var b bytes.Buffer
		if err := tpl.Execute(&b, vars); err != nil {
			return nil, errf(400, err.Error())
		}
		return map[string]any{"ok": true, "html": b.String()}, nil
	}
}

func (s *Service) republishTemplate(ctx context.Context, id uint64) {
	if s.republish == nil {
		return
	}
	ids, err := s.sites.SiteIDsByTemplate(ctx, id)
	if err != nil || len(ids) == 0 {
		return
	}
	s.republish(ids)
}

func (s *Service) pruneTemplateFiles(ctx context.Context, id uint64) {
	files, err := s.sites.ListTemplateFiles(ctx, id)
	if err != nil || len(files) <= 5 {
		return
	}
	for _, f := range files[5:] {
		_ = s.sites.DeleteTemplateFile(ctx, f.ID)
		_ = os.RemoveAll(f.Dir)
	}
}

func templateRow(t *domainsite.Template, withBody bool, used int64, files []domainsite.TemplateFile) map[string]any {
	kind := t.Kind
	if kind == "" {
		kind = "html"
	}
	status := t.Status
	if status == "" {
		status = "enabled"
	}
	row := map[string]any{
		"templateId": t.ID, "code": t.Code, "name": t.Name, "description": deref(t.Description),
		"kind": kind, "status": status, "url": t.URL, "hostHeader": t.HostHeader, "timeoutMs": t.TimeoutMs,
		"entry": t.Entry, "spaFallback": t.SpaFallback, "activeVersion": t.ActiveVersion, "usedCount": used,
		"createdAt": t.CreatedAt.Format("2006-01-02 15:04:05"),
		"updatedAt": t.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if withBody && kind == "html" {
		row["html"] = t.HTML
	}
	if files != nil {
		vers := make([]map[string]any, 0, len(files))
		for _, f := range files {
			item := map[string]any{
				"version": f.Version, "entry": f.Entry, "bytes": f.Bytes, "fileCount": f.FileCount,
				"createdAt": f.CreatedAt.Format("2006-01-02 15:04:05"), "current": f.Version == t.ActiveVersion,
			}
			if withBody && f.Version == t.ActiveVersion {
				item["files"] = listTopFiles(f.Dir)
			}
			vers = append(vers, item)
		}
		row["versions"] = vers
	}
	return row
}

func listTopFiles(dir string) []string {
	var names []string
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || len(names) >= 40 {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil || rel == "." {
			return nil
		}
		depth := strings.Count(filepath.ToSlash(rel), "/")
		if depth > 1 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		name := filepath.ToSlash(rel)
		if d.IsDir() {
			name += "/"
		}
		names = append(names, name)
		return nil
	})
	return names
}

func unzipTemplate(r io.ReaderAt, size int64, dir, entry string) (string, int, int64, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return "", 0, 0, errf(400, "请上传 zip 压缩包")
	}
	if len(zr.File) > 2000 {
		return "", 0, 0, errf(400, "压缩包内文件不能超过 2000 个")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", 0, 0, err
	}
	var count int
	var total int64
	for _, f := range zr.File {
		name := path.Clean("/" + strings.ReplaceAll(f.Name, "\\", "/"))
		name = strings.TrimPrefix(name, "/")
		if name == "" || strings.HasPrefix(name, "../") || path.IsAbs(f.Name) {
			return "", 0, 0, errf(400, "压缩包包含非法路径")
		}
		if f.Mode()&os.ModeSymlink != 0 {
			return "", 0, 0, errf(400, "压缩包不能包含符号链接")
		}
		target := filepath.Join(dir, filepath.FromSlash(name))
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return "", 0, 0, err
			}
			continue
		}
		if f.UncompressedSize64 > 20<<20 {
			return "", 0, 0, errf(400, "单个文件不能超过 20MB")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", 0, 0, err
		}
		rc, err := f.Open()
		if err != nil {
			return "", 0, 0, err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			rc.Close()
			return "", 0, 0, err
		}
		n, copyErr := io.Copy(out, io.LimitReader(rc, 20<<20+1))
		out.Close()
		rc.Close()
		if copyErr != nil {
			return "", 0, 0, copyErr
		}
		if n > 20<<20 {
			return "", 0, 0, errf(400, "单个文件不能超过 20MB")
		}
		count++
		total += n
		if total > 80<<20 {
			return "", 0, 0, errf(400, "解压后总大小不能超过 80MB")
		}
	}
	entry = cleanEntry(entry)
	if entry == "index.html" {
		if _, err := os.Stat(filepath.Join(dir, "index.html")); err != nil {
			if _, err2 := os.Stat(filepath.Join(dir, "index.htm")); err2 == nil {
				entry = "index.htm"
			}
		}
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(entry))); err != nil {
		return "", 0, 0, errf(400, "压缩包中找不到入口文件 "+entry)
	}
	return entry, count, total, nil
}

func cleanEntry(entry string) string {
	entry = path.Clean("/" + strings.TrimSpace(entry))
	entry = strings.TrimPrefix(entry, "/")
	if entry == "" || entry == "." {
		return "index.html"
	}
	return entry
}

func validatePublicURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return errf(400, "请填写 http 或 https 网址")
	}
	host := u.Hostname()
	if ip := net.ParseIP(host); ip != nil {
		if !publicIP(ip) {
			return errf(400, "不允许指向本机或内网地址")
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return errf(400, "无法解析该网址")
	}
	for _, ip := range ips {
		if !publicIP(ip) {
			return errf(400, "不允许指向本机或内网地址")
		}
	}
	return nil
}

func publicIP(ip net.IP) bool {
	return ip != nil && !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsUnspecified()
}

func probeURL(raw, host string, timeout time.Duration) (string, error) {
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	host = probe.NormalizeHost(host)
	client := &http.Client{Timeout: timeout, Transport: probe.RoundTripper(raw, host), CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 3 {
			return http.ErrUseLastResponse
		}
		return nil
	}}
	req, err := http.NewRequest(http.MethodHead, raw, nil)
	if err != nil {
		return "", err
	}
	if host != "" {
		req.Host = host
	}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotFound {
		if resp != nil {
			resp.Body.Close()
		}
		req, err2 := http.NewRequest(http.MethodGet, raw, nil)
		if err2 != nil {
			return "", err
		}
		if host != "" {
			req.Host = host
		}
		resp, err = client.Do(req)
	}
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 32<<10))
	return "探测完成，状态码 " + itoa(uint64(resp.StatusCode)) + "，耗时 " + time.Since(start).Truncate(time.Millisecond).String(), nil
}

func itoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
