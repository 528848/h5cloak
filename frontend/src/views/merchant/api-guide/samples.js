/**
 * 接入示例代码。{{BASE}} 网关地址，{{PREFIX}} 密钥前 12 位，{{HOST}} 商户域名。
 * 签名：hex(HMAC-SHA256(secret, ts\nnonce\nMETHOD\npath\nhex(sha256(body))))
 */
export const languages = [
  { key: 'php', label: 'PHP' },
  { key: 'node', label: 'Node.js' },
  { key: 'go', label: 'Go' },
  { key: 'python', label: 'Python' }
];

export const signSamples = {
  php: `<?php
function cloakRequest(string $method, string $path, array $body = null): array {
    $secret = getenv('CLOAK_SECRET');          // sk_... 完整密钥，只放在服务端
    $base   = '{{BASE}}';
    $json   = $body === null ? '' : json_encode($body, JSON_UNESCAPED_SLASHES);
    $ts     = (string) time();
    $nonce  = bin2hex(random_bytes(16));
    $msg    = $ts . "\\n" . $nonce . "\\n" . strtoupper($method) . "\\n" . $path . "\\n" . hash('sha256', $json);
    $sig    = hash_hmac('sha256', $msg, $secret);

    $ch = curl_init($base . $path);
    curl_setopt_array($ch, [
        CURLOPT_CUSTOMREQUEST => strtoupper($method),
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_TIMEOUT => 3,
        CURLOPT_HTTPHEADER => [
            'Content-Type: application/json',
            'X-Cloak-Key: ' . substr($secret, 0, 12),
            'X-Cloak-Timestamp: ' . $ts,
            'X-Cloak-Nonce: ' . $nonce,
            'X-Cloak-Signature: ' . $sig,
        ],
        CURLOPT_POSTFIELDS => $json === '' ? null : $json,
    ]);
    $raw = curl_exec($ch);
    curl_close($ch);
    return json_decode($raw, true) ?: ['code' => 0];
}`,
  node: `import crypto from 'node:crypto';

const BASE = '{{BASE}}';
const SECRET = process.env.CLOAK_SECRET; // sk_... 完整密钥，只放在服务端

export async function cloakRequest(method, path, body) {
  const json = body === undefined ? '' : JSON.stringify(body);
  const ts = String(Math.floor(Date.now() / 1000));
  const nonce = crypto.randomBytes(16).toString('hex');
  const bodyHash = crypto.createHash('sha256').update(json).digest('hex');
  const msg = [ts, nonce, method.toUpperCase(), path, bodyHash].join('\\n');
  const sig = crypto.createHmac('sha256', SECRET).update(msg).digest('hex');
  const res = await fetch(BASE + path, {
    method,
    headers: {
      'Content-Type': 'application/json',
      'X-Cloak-Key': SECRET.slice(0, 12),
      'X-Cloak-Timestamp': ts,
      'X-Cloak-Nonce': nonce,
      'X-Cloak-Signature': sig
    },
    body: json || undefined,
    signal: AbortSignal.timeout(3000)
  });
  return res.json();
}`,
  go: `package cloak

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const base = "{{BASE}}"

var secret = os.Getenv("CLOAK_SECRET") // sk_... 完整密钥，只放在服务端

func Request(method, path string, body any) (*http.Response, error) {
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nb := make([]byte, 16)
	_, _ = rand.Read(nb)
	nonce := hex.EncodeToString(nb)
	sum := sha256.Sum256(raw)
	msg := strings.Join([]string{ts, nonce, strings.ToUpper(method), path, hex.EncodeToString(sum[:])}, "\\n")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))

	req, _ := http.NewRequest(method, base+path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cloak-Key", secret[:12])
	req.Header.Set("X-Cloak-Timestamp", ts)
	req.Header.Set("X-Cloak-Nonce", nonce)
	req.Header.Set("X-Cloak-Signature", hex.EncodeToString(mac.Sum(nil)))
	return (&http.Client{Timeout: 3 * time.Second}).Do(req)
}`,
  python: `import hashlib, hmac, json, os, secrets, time
import requests

BASE = "{{BASE}}"
SECRET = os.environ["CLOAK_SECRET"]  # sk_... 完整密钥，只放在服务端

def cloak_request(method: str, path: str, body: dict | None = None) -> dict:
    raw = "" if body is None else json.dumps(body, separators=(",", ":"))
    ts = str(int(time.time()))
    nonce = secrets.token_hex(16)
    msg = "\\n".join([ts, nonce, method.upper(), path, hashlib.sha256(raw.encode()).hexdigest()])
    sig = hmac.new(SECRET.encode(), msg.encode(), hashlib.sha256).hexdigest()
    resp = requests.request(method, BASE + path, data=raw or None, timeout=3, headers={
        "Content-Type": "application/json",
        "X-Cloak-Key": SECRET[:12],
        "X-Cloak-Timestamp": ts,
        "X-Cloak-Nonce": nonce,
        "X-Cloak-Signature": sig,
    })
    return resp.json()`
};

export const scenarios = [
  {
    key: 'decision',
    title: '场景一：页面请求判定',
    desc: '访客每次打开页面时，服务端把 URL、Cookie、IP、UA 发给判定接口；按 content 决定输出真实页面还是审核内容，按 cookie 指令写入访客令牌。地址栏保持不变。',
    endpoint: 'POST /api/open/v2/decision',
    code: {
      php: `$res = cloakRequest('POST', '/api/open/v2/decision', [
    'url'          => (empty($_SERVER['HTTPS']) ? 'http://' : 'https://') . $_SERVER['HTTP_HOST'] . $_SERVER['REQUEST_URI'],
    'visitorToken' => $_COOKIE['_ck_vid'] ?? '',
    'ip'           => $_SERVER['REMOTE_ADDR'],
    'userAgent'    => $_SERVER['HTTP_USER_AGENT'] ?? '',
    'referer'      => $_SERVER['HTTP_REFERER'] ?? '',
]);
$d = $res['data'] ?? ['content' => 'review'];   // 接口异常时按兜底处理

if (!empty($d['cookie'])) {
    $c = $d['cookie'];
    setcookie($c['name'], $c['value'], [
        'expires' => time() + $c['maxAge'], 'path' => '/', 'domain' => $c['domain'] ?: '',
        'secure' => $c['secure'], 'httponly' => true, 'samesite' => 'Lax',
    ]);
}
if ($d['content'] !== 'real' && $d['content'] !== 'bypass') {
    header('Cache-Control: no-store');
    header('X-Robots-Tag: noindex');
    if (($d['renderHint']['type'] ?? '') === 'html') { echo $d['renderHint']['html']; exit; }
    include __DIR__ . '/review.html';                 // 或渲染站内合规页面
    exit;
}
// 放行：继续输出真实页面`,
      node: `app.use(async (req, res, next) => {
  let d = { content: 'review' };
  try {
    const out = await cloakRequest('POST', '/api/open/v2/decision', {
      url: \`\${req.protocol}://\${req.get('host')}\${req.originalUrl}\`,
      visitorToken: req.cookies._ck_vid || '',
      ip: req.ip,
      userAgent: req.get('user-agent') || '',
      referer: req.get('referer') || ''
    });
    d = out.data || d;
  } catch (e) { /* 超时或网络错误：按兜底（审核）处理 */ }

  if (d.cookie) {
    res.cookie(d.cookie.name, d.cookie.value, {
      maxAge: d.cookie.maxAge * 1000, domain: d.cookie.domain || undefined,
      secure: d.cookie.secure, httpOnly: true, sameSite: 'lax', path: '/'
    });
  }
  if (d.content !== 'real' && d.content !== 'bypass') {
    res.set({ 'Cache-Control': 'no-store', 'X-Robots-Tag': 'noindex' });
    if (d.renderHint?.type === 'html') return res.send(d.renderHint.html);
    return res.render('review');
  }
  next();
});`,
      go: `func cloakMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scheme := "https"
		if r.TLS == nil { scheme = "http" }
		tok := ""
		if c, err := r.Cookie("_ck_vid"); err == nil { tok = c.Value }

		content := "review"
		var data map[string]any
		resp, err := Request("POST", "/api/open/v2/decision", map[string]any{
			"url": scheme + "://" + r.Host + r.RequestURI, "visitorToken": tok,
			"ip": r.RemoteAddr, "userAgent": r.UserAgent(), "referer": r.Referer(),
		})
		if err == nil {
			var out struct{ Data map[string]any }
			_ = json.NewDecoder(resp.Body).Decode(&out)
			resp.Body.Close()
			data = out.Data
			if c, _ := data["content"].(string); c != "" { content = c }
		}
		if ck, ok := data["cookie"].(map[string]any); ok {
			http.SetCookie(w, &http.Cookie{Name: ck["name"].(string), Value: ck["value"].(string),
				MaxAge: int(ck["maxAge"].(float64)), Path: "/", HttpOnly: true, Secure: ck["secure"].(bool), SameSite: http.SameSiteLaxMode})
		}
		if content != "real" && content != "bypass" {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("X-Robots-Tag", "noindex")
			if h, ok := data["renderHint"].(map[string]any); ok && h["type"] == "html" {
				w.Write([]byte(h["html"].(string)))
				return
			}
			http.ServeFile(w, r, "review.html")
			return
		}
		next.ServeHTTP(w, r)
	})
}`,
      python: `@app.before_request
def cloak_gate():
    try:
        out = cloak_request("POST", "/api/open/v2/decision", {
            "url": request.url,
            "visitorToken": request.cookies.get("_ck_vid", ""),
            "ip": request.remote_addr,
            "userAgent": request.headers.get("User-Agent", ""),
            "referer": request.headers.get("Referer", ""),
        })
        d = out.get("data") or {"content": "review"}
    except Exception:
        d = {"content": "review"}          # 超时或网络错误：按兜底处理

    g.cloak = d
    if d["content"] not in ("real", "bypass"):
        hint = d.get("renderHint") or {}
        body = hint["html"] if hint.get("type") == "html" else render_template("review.html")
        resp = make_response(body, 200)
        resp.headers["Cache-Control"] = "no-store"
        resp.headers["X-Robots-Tag"] = "noindex"
        return _set_cookie(resp, d)

@app.after_request
def cloak_cookie(resp):
    return _set_cookie(resp, getattr(g, "cloak", {}))

def _set_cookie(resp, d):
    c = d.get("cookie")
    if c:
        resp.set_cookie(c["name"], c["value"], max_age=c["maxAge"], domain=c["domain"] or None,
                        secure=c["secure"], httponly=True, samesite="Lax", path="/")
    return resp`
    }
  },
  {
    key: 'capture',
    title: '场景二：只做归因不改内容',
    desc: '落地页本身不需要切换内容，只想记录访客来自哪条链接并种下令牌，之后在下单时查归因。',
    endpoint: 'POST /api/open/v2/attribution/capture',
    code: {
      php: `$res = cloakRequest('POST', '/api/open/v2/attribution/capture', [
    'url' => 'https://{{HOST}}' . $_SERVER['REQUEST_URI'],
    'visitorToken' => $_COOKIE['_ck_vid'] ?? '',
    'ip' => $_SERVER['REMOTE_ADDR'], 'userAgent' => $_SERVER['HTTP_USER_AGENT'] ?? '',
]);
if (!empty($res['data']['cookie'])) { /* 同场景一写 cookie */ }
// $res['data']['attributed'], ['channel'], ['code']['code'] 可直接记到你的会话里`,
      node: `const { data } = await cloakRequest('POST', '/api/open/v2/attribution/capture', {
  url: 'https://{{HOST}}' + req.originalUrl,
  visitorToken: req.cookies._ck_vid || '',
  ip: req.ip, userAgent: req.get('user-agent') || ''
});
if (data.cookie) { /* 同场景一写 cookie */ }
req.session.channel = data.channel;   // 如 distributor:ALICE`,
      go: `resp, err := Request("POST", "/api/open/v2/attribution/capture", map[string]any{
	"url": "https://{{HOST}}" + r.RequestURI, "visitorToken": tok,
	"ip": r.RemoteAddr, "userAgent": r.UserAgent(),
})
// data.cookie 同场景一写入；data.channel / data.code 记入会话`,
      python: `out = cloak_request("POST", "/api/open/v2/attribution/capture", {
    "url": "https://{{HOST}}" + request.full_path,
    "visitorToken": request.cookies.get("_ck_vid", ""),
    "ip": request.remote_addr, "userAgent": request.headers.get("User-Agent", ""),
})
session["channel"] = out["data"].get("channel")`
    }
  },
  {
    key: 'resolve',
    title: '场景三：下单时查归因',
    desc: '结账或创建订单时，用访客 Cookie 查出这位访客归属哪个分销码、哪个渠道，写进订单。',
    endpoint: 'POST /api/open/v2/attribution/resolve',
    code: {
      php: `$res = cloakRequest('POST', '/api/open/v2/attribution/resolve', [
    'visitorToken' => $_COOKIE['_ck_vid'] ?? '',
]);
$attr = $res['data'] ?? [];
if (!empty($attr['found'])) {
    $order->channel   = $attr['channel'];            // distributor:ALICE
    $order->promoCode = $attr['code']['code'] ?? null;
}`,
      node: `const { data } = await cloakRequest('POST', '/api/open/v2/attribution/resolve', {
  visitorToken: req.cookies._ck_vid || ''
});
if (data.found) {
  order.channel = data.channel;
  order.promoCode = data.code?.code ?? null;
}`,
      go: `resp, _ := Request("POST", "/api/open/v2/attribution/resolve", map[string]any{"visitorToken": tok})
var out struct{ Data struct {
	Found   bool   \`json:"found"\`
	Channel string \`json:"channel"\`
	Code    *struct{ Code string \`json:"code"\` } \`json:"code"\`
} }
_ = json.NewDecoder(resp.Body).Decode(&out)
if out.Data.Found { order.Channel = out.Data.Channel }`,
      python: `attr = cloak_request("POST", "/api/open/v2/attribution/resolve", {
    "visitorToken": request.cookies.get("_ck_vid", "")
})["data"]
if attr.get("found"):
    order.channel = attr["channel"]
    order.promo_code = (attr.get("code") or {}).get("code")`
    }
  },
  {
    key: 'conversion',
    title: '场景四：支付成功后回传转化',
    desc: '支付成功回调里把订单号、金额和访客 Cookie（或邮箱）回传，系统会累计到对应分销码的转化数，并把同一邮箱的不同访客合并。接口幂等，重复回传同一订单号不会重复计数。',
    endpoint: 'POST /api/open/v2/conversions',
    code: {
      php: `cloakRequest('POST', '/api/open/v2/conversions', [
    'orderNo'      => $order->no,
    'status'       => 'paid',
    'amount'       => $order->amount,
    'currency'     => 'USD',
    'visitorToken' => $order->visitorToken,        // 下单时保存的 _ck_vid
    'email'        => $order->email,               // 服务端只保存哈希
    'occurredAt'   => date(DATE_ATOM),
]);`,
      node: `await cloakRequest('POST', '/api/open/v2/conversions', {
  orderNo: order.no, status: 'paid', amount: order.amount, currency: 'USD',
  visitorToken: order.visitorToken, email: order.email,
  occurredAt: new Date().toISOString()
});`,
      go: `_, _ = Request("POST", "/api/open/v2/conversions", map[string]any{
	"orderNo": order.No, "status": "paid", "amount": order.Amount, "currency": "USD",
	"visitorToken": order.VisitorToken, "email": order.Email,
	"occurredAt": time.Now().UTC().Format(time.RFC3339),
})`,
      python: `cloak_request("POST", "/api/open/v2/conversions", {
    "orderNo": order.no, "status": "paid", "amount": float(order.amount), "currency": "USD",
    "visitorToken": order.visitor_token, "email": order.email,
    "occurredAt": datetime.utcnow().isoformat() + "Z",
})`
    }
  }
];

export const v1Sample = `# V1 插件 / 旧接入方式仍可用：直接带完整密钥，不需要签名
curl -X POST {{BASE}}/api/open/decision \\
  -H "X-API-Key: sk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" \\
  -H "Content-Type: application/json" \\
  -d '{"host":"{{HOST}}","path":"/lp","query":"ref=ALICE","ip":"203.0.113.10","userAgent":"Mozilla/5.0"}'
# 返回 data.page = real | review（兼容字段），同时含 v2 的 content / cookie / renderHint`;

export const errorCodes = [
  ['missing_credentials', '缺少签名头或 X-API-Key'],
  ['missing_signature', '缺少 X-Cloak-Timestamp / Nonce / Signature 之一'],
  ['timestamp_expired', '时间戳与服务器相差超过 300 秒，检查服务器时钟'],
  ['bad_nonce', 'nonce 长度须 8–64'],
  ['nonce_replayed', '同一 nonce 5 分钟内重复使用'],
  [
    'bad_signature',
    '签名不匹配：检查 path 是否含查询串、body 是否与签名时完全一致'
  ],
  ['unknown_key', '密钥前缀不存在，站点可能已删除或密钥已重置'],
  ['invalid_api_key', 'V1 方式密钥错误'],
  ['site_disabled', '站点或商户已停用'],
  [
    'secret_unavailable',
    '密钥创建于启用签名之前，请在站点配置页重置一次服务端密钥'
  ],
  ['ip_not_allowed', '来源 IP 不在站点密钥 IP 白名单'],
  ['rate_limited', '超过每密钥 50 次/秒 或每站点 2000 次/分']
];
