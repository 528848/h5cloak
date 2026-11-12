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
  var bannerMode = script.getAttribute('data-banner') || '';
  var bannerOff = bannerMode === 'off';
  var presetOnly = bannerMode === 'preset';
  if ((!key || !endpoint) && !presetOnly) return;

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
  // 点击后交给系统默认浏览器。Android 用 BROWSABLE，不指定 Chrome。
  function openInBrowser(url, os) {
    if (!url) return '';
    if (os === 'ios' && url.indexOf('https://') === 0) return 'x-safari-' + url;
    if (os === 'android') {
      try {
        var u = new URL(url);
        var scheme = (u.protocol || 'https:').replace(':', '');
        return 'intent://' + u.host + u.pathname + u.search + '#Intent;scheme=' + scheme + ';action=android.intent.action.VIEW;category=android.intent.category.BROWSABLE;S.browser_fallback_url=' + encodeURIComponent(url) + ';end';
      } catch (e) {}
    }
    return url;
  }
  function banner(d) {
    if (bannerOff || !d || !d.handoffUrl || !d.handoffBanner || !d.handoffBanner.enabled) return;
    if (getCookie('_ck_hb') || document.getElementById('ck-handoff')) return;
    var os = (d.inAppBrowser && d.inAppBrowser.os) || '';
    var href = openInBrowser(d.handoffUrl, os);
    var box = document.createElement('div');
    box.id = 'ck-handoff';
    box.style.cssText = 'position:fixed;z-index:2147483647;left:50%;top:18%;transform:translateX(-50%);width:min(360px,calc(100% - 32px));background:#fff;color:#1f2329;border-radius:12px;box-shadow:0 16px 48px rgba(0,0,0,.22);font:14px/1.5 -apple-system,BlinkMacSystemFont,sans-serif;padding:16px 16px 14px;box-sizing:border-box';
    var x = document.createElement('button'); x.type = 'button'; x.textContent = '×'; x.setAttribute('aria-label', 'close');
    x.style.cssText = 'position:absolute;top:8px;right:10px;background:none;border:0;color:#8a8f99;font-size:20px;line-height:1;padding:0;cursor:pointer';
    x.onclick = function () { box.style.display = 'none'; document.cookie = '_ck_hb=1;path=/;SameSite=Lax'; };
    var text = document.createElement('div'); text.style.margin = '0 22px 14px 0'; text.textContent = d.handoffBanner.text || '在系统浏览器中打开';
    var a = document.createElement('a'); a.href = href; a.textContent = '打开';
    a.style.cssText = 'display:inline-block;background:#1677ff;color:#fff;border-radius:6px;padding:8px 14px;text-decoration:none;font-weight:600';
    box.appendChild(x); box.appendChild(text); box.appendChild(a);
    if (os === 'ios') {
      var hint = document.createElement('div');
      hint.style.cssText = 'margin-top:8px;color:#8a8f99;font-size:12px';
      hint.textContent = '如果没有离开本页，请点右上角菜单，选择在 Safari 中打开。';
      box.appendChild(hint);
    }
    (document.body || document.documentElement).appendChild(box);
  }
  function showPreset() {
    var p = window.__CLOAK_HANDOFF__;
    if (!p || !p.url || p.enabled === false) return;
    banner({ handoffUrl: p.url, handoffBanner: { enabled: true, text: p.text || '' }, inAppBrowser: { os: p.os || '', detected: true } });
  }
  function fire(name, detail) {
    // bubbles 让 document 与 window 上的监听都能收到
    try { document.dispatchEvent(new CustomEvent(name, { detail: detail, bubbles: true })); } catch (e) {}
  }

  window.Cloak = window.Cloak || {};
  window.Cloak.getVisitorToken = window.Cloak.getVisitorToken || function () { return getCookie(cookieName); };
  window.Cloak.show = window.Cloak.show || show;
  window.Cloak.openInBrowser = openInBrowser;
  window.Cloak.showHandoff = banner;
  if (presetOnly) {
    if (document.body) showPreset();
    else document.addEventListener('DOMContentLoaded', showPreset);
    return;
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
})();
