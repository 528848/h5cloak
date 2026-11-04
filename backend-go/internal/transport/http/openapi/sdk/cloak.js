/*! Cloak JS SDK v2 — 归因上报与内容区块切换。
 * 用法：
 *   <script src="https://<网关>/_cloak/sdk.js" data-key="pk_xxx" data-endpoint="https://<网关>" async></script>
 *   <div data-cloak="real" hidden>…真实内容…</div>
 *   <div data-cloak="review">…审核内容…</div>
 * 可选：data-cookie="_ck_vid" 自定义 cookie 名；data-banner="off" 关闭引导条。
 * 事件：document 上触发 "cloak:decision"，detail 为接口返回的 data。
 */
(function () {
  'use strict';
  var script = document.currentScript || (function () {
    var s = document.getElementsByTagName('script');
    for (var i = s.length - 1; i >= 0; i--) if (s[i].getAttribute('data-key')) return s[i];
    return null;
  })();
  if (!script) return;
  var key = script.getAttribute('data-key');
  var endpoint = (script.getAttribute('data-endpoint') || (function () {
    try { var u = new URL(script.src); return u.origin; } catch (e) { return ''; }
  })()).replace(/\/+$/, '');
  var cookieName = script.getAttribute('data-cookie') || '_ck_vid';
  var bannerOff = script.getAttribute('data-banner') === 'off';
  if (!key || !endpoint) return;

  function getCookie(name) {
    var m = document.cookie.match(new RegExp('(?:^|; )' + name.replace(/[$()*+.?[\\\]^{|}]/g, '\\$&') + '=([^;]*)'));
    return m ? decodeURIComponent(m[1]) : '';
  }
  function setCookie(name, value, maxAge, domain) {
    var c = name + '=' + encodeURIComponent(value) + '; Path=/; Max-Age=' + maxAge + '; SameSite=Lax';
    if (domain) c += '; Domain=' + domain;
    if (location.protocol === 'https:') c += '; Secure';
    document.cookie = c;
  }
  function show(kind) {
    var nodes = document.querySelectorAll('[data-cloak]');
    for (var i = 0; i < nodes.length; i++) {
      var want = nodes[i].getAttribute('data-cloak');
      var on = want === kind || (kind === 'bypass' && want === 'real');
      if (on) { nodes[i].removeAttribute('hidden'); nodes[i].style.removeProperty('display'); }
      else { nodes[i].setAttribute('hidden', ''); }
    }
    document.documentElement.setAttribute('data-cloak-content', kind);
  }
  function banner(d) {
    if (bannerOff || !d.handoffUrl || !d.handoffBanner || !d.handoffBanner.enabled) return;
    if (getCookie('_ck_hb')) return;
    var href = d.handoffUrl;
    var os = (d.inAppBrowser && d.inAppBrowser.os) || '';
    if (os === 'ios' && href.indexOf('https://') === 0) href = 'x-safari-' + href;
    else if (os === 'android') { try { var u = new URL(href); href = 'intent://' + u.host + u.pathname + u.search + '#Intent;scheme=' + u.protocol.replace(':', '') + ';action=android.intent.action.VIEW;end'; } catch (e) {} }
    var box = document.createElement('div');
    box.id = 'ck-handoff';
    box.style.cssText = 'position:fixed;left:0;right:0;bottom:0;z-index:2147483647;background:#111;color:#fff;font:14px/1.4 -apple-system,BlinkMacSystemFont,sans-serif;padding:12px 14px;display:flex;align-items:center;gap:12px;box-shadow:0 -2px 10px rgba(0,0,0,.25)';
    var text = document.createElement('span'); text.style.flex = '1'; text.textContent = d.handoffBanner.text || '在浏览器中打开，继续访问';
    var a = document.createElement('a'); a.href = href; a.textContent = '打开';
    a.style.cssText = 'background:#fff;color:#111;border-radius:6px;padding:8px 12px;text-decoration:none;font-weight:600;white-space:nowrap';
    var x = document.createElement('button'); x.type = 'button'; x.textContent = '×'; x.setAttribute('aria-label', 'close');
    x.style.cssText = 'background:none;border:0;color:#aaa;font-size:20px;line-height:1;padding:0 4px';
    x.onclick = function () { box.style.display = 'none'; document.cookie = '_ck_hb=1;path=/;SameSite=Lax'; };
    box.appendChild(text); box.appendChild(a); box.appendChild(x);
    (document.body || document.documentElement).appendChild(box);
  }
  function fire(name, detail) {
    // bubbles 让 document 与 window 上的监听都能收到
    try { document.dispatchEvent(new CustomEvent(name, { detail: detail, bubbles: true })); } catch (e) {}
  }

  var payload = JSON.stringify({ key: key, url: location.href, referrer: document.referrer || '', visitorToken: getCookie(cookieName) });
  var done = false;
  function apply(d) {
    if (done) return; done = true;
    if (d && d.cookie && d.visitorToken) setCookie(d.cookie.name || cookieName, d.visitorToken, d.cookie.maxAge || 2592000, d.cookie.domain || '');
    var kind = (d && d.content) || 'review';
    if (document.body) { show(kind); banner(d || {}); }
    else document.addEventListener('DOMContentLoaded', function () { show(kind); banner(d || {}); });
    fire('cloak:decision', d || { content: 'review', failed: true });
  }
  var ctl = typeof AbortController !== 'undefined' ? new AbortController() : null;
  var timer = setTimeout(function () { if (ctl) ctl.abort(); apply(null); }, 3000);
  fetch(endpoint + '/api/open/v2/js/capture', {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: payload,
    credentials: 'omit', signal: ctl ? ctl.signal : undefined, keepalive: true
  }).then(function (r) { return r.json(); })
    .then(function (j) { clearTimeout(timer); apply(j && j.data ? j.data : null); })
    .catch(function () { clearTimeout(timer); apply(null); });

  window.Cloak = window.Cloak || {
    getVisitorToken: function () { return getCookie(cookieName); },
    show: show
  };
})();
