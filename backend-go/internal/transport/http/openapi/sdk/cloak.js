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
  function intOr(v, dflt, min, max) {
    if (typeof v !== 'number' || !isFinite(v)) return dflt;
    v = Math.round(v);
    if (v < min) return min;
    if (v > max) return max;
    return v;
  }
  function readGuide() {
    var raw = getCookie('_ck_hg');
    var parts = raw ? raw.split('.') : [];
    if (parts.length !== 4 || parts[0] !== 'v1') return { shows: 0, next: 0, left: 0 };
    return {
      shows: parseInt(parts[1], 10) || 0,
      next: parseInt(parts[2], 10) || 0,
      left: parseInt(parts[3], 10) || 0
    };
  }
  function writeGuide(g, maxAge) {
    setCookie('_ck_hg', 'v1.' + g.shows + '.' + g.next + '.' + (g.left ? 1 : 0), maxAge || 604800, '');
  }
  function pickText(primary, other, key, fallback) {
    primary = primary || {};
    other = other || {};
    return primary[key] || other[key] || fallback || '';
  }
  function copyFor(cfg) {
    var narrow = !window.matchMedia || window.matchMedia('(max-width: 767px)').matches;
    var primary = narrow ? cfg.mobile : cfg.desktop;
    var other = narrow ? cfg.desktop : cfg.mobile;
    return {
      narrow: narrow,
      title: pickText(primary, other, 'title', ''),
      body: pickText(primary, other, 'body', cfg.text || ''),
      open: pickText(primary, other, 'open', '打开'),
      stay: pickText(primary, other, 'stay', '继续浏览')
    };
  }
  function placeBox(box, narrow) {
    box.style.cssText = narrow
      ? 'position:fixed;z-index:2147483647;left:12px;right:12px;bottom:calc(12px + env(safe-area-inset-bottom));max-height:40vh;overflow:auto;background:#fff;color:#1f2329;border-radius:12px;box-shadow:0 16px 48px rgba(0,0,0,.22);font:14px/1.5 -apple-system,BlinkMacSystemFont,sans-serif;padding:16px 16px 14px;box-sizing:border-box'
      : 'position:fixed;z-index:2147483647;right:24px;bottom:24px;width:360px;background:#fff;color:#1f2329;border-radius:12px;box-shadow:0 16px 48px rgba(0,0,0,.22);font:14px/1.5 -apple-system,BlinkMacSystemFont,sans-serif;padding:16px 16px 14px;box-sizing:border-box';
  }
  function schedule(cfg) {
    if (bannerOff || !cfg || !cfg.url || window.__CLOAK_GUIDE_ARMED) return;
    if (getCookie('_ck_hb')) return;
    var guide = readGuide();
    if (guide.left) return;
    var maxShows = intOr(cfg.maxShows, 3, 1, 5);
    if (guide.shows >= maxShows) return;
    window.__CLOAK_GUIDE_ARMED = true;
    var delayMs = intOr(cfg.delaySeconds, 10, 0, 120) * 1000;
    var coolSec = intOr(cfg.cooldownMinutes, 30, 5, 1440) * 60;
    var maxAge = cfg.maxAge > 0 ? cfg.maxAge : 604800;
    if (maxAge > 2592000) maxAge = 2592000;
    var timer = null;
    var elapsed = 0;
    var tickAt = 0;
    var painted = false;
    var box = null;
    function clearTimer() {
      if (timer) clearTimeout(timer);
      timer = null;
    }
    function visible() {
      return !document.visibilityState || document.visibilityState === 'visible';
    }
    function stopClock() {
      if (tickAt) elapsed += Date.now() - tickAt;
      tickAt = 0;
      clearTimer();
    }
    function paint() {
      if (painted || document.getElementById('ck-handoff')) return;
      painted = true;
      var nowSec = Math.floor(Date.now() / 1000);
      guide.shows += 1;
      guide.next = nowSec + coolSec;
      writeGuide(guide, maxAge);
      box = document.createElement('div');
      box.id = 'ck-handoff';
      var title = document.createElement('div');
      title.style.cssText = 'font-weight:600;margin:0 22px 6px 0';
      var body = document.createElement('div');
      body.style.cssText = 'margin:0 22px 14px 0';
      var hint = document.createElement('div');
      hint.style.cssText = 'margin-top:8px;color:#8a8f99;font-size:12px;display:none';
      var row = document.createElement('div');
      var open = document.createElement('a');
      open.href = cfg.url;
      open.style.cssText = 'display:inline-block;background:#1677ff;color:#fff;border-radius:6px;padding:8px 14px;text-decoration:none;font-weight:600;cursor:pointer';
      var stay = document.createElement('button');
      stay.type = 'button';
      stay.style.cssText = 'margin-left:8px;background:none;border:0;color:#4e5969;padding:8px 4px;cursor:pointer';
      var x = document.createElement('button');
      x.type = 'button';
      x.textContent = '×';
      x.setAttribute('aria-label', '关闭');
      x.style.cssText = 'position:absolute;top:8px;right:10px;background:none;border:0;color:#8a8f99;font-size:20px;line-height:1;padding:0;cursor:pointer';
      function render() {
        var copy = copyFor(cfg);
        placeBox(box, copy.narrow);
        title.textContent = copy.title;
        title.style.display = copy.title ? '' : 'none';
        body.textContent = copy.body;
        open.textContent = copy.open;
        stay.textContent = copy.stay;
      }
      function dismiss() {
        if (box.parentNode) box.parentNode.removeChild(box);
        if (cfg.dismissEnds) {
          guide.shows = maxShows;
          writeGuide(guide, maxAge);
        }
      }
      function markLeft() {
        guide.left = 1;
        writeGuide(guide, maxAge);
        if (box.parentNode) box.parentNode.removeChild(box);
      }
      x.onclick = dismiss;
      stay.onclick = dismiss;
      open.onclick = function (ev) {
        if (ev && ev.preventDefault) ev.preventDefault();
        var jumped = false;
        function left() {
          if (jumped) return;
          jumped = true;
          markLeft();
        }
        var wait = setTimeout(function () {
          if (!jumped && visible() && cfg.os === 'ios') {
            hint.style.display = '';
            hint.textContent = '如果没有离开本页，请点右上角菜单，选择在 Safari 中打开。';
          }
        }, 2000);
        document.addEventListener('visibilitychange', function () {
          if (document.visibilityState === 'hidden') { clearTimeout(wait); left(); }
        });
        window.addEventListener('pagehide', function () { clearTimeout(wait); left(); });
        location.href = openInBrowser(cfg.url, cfg.os);
      };
      row.appendChild(open);
      row.appendChild(stay);
      box.appendChild(x);
      box.appendChild(title);
      box.appendChild(body);
      box.appendChild(row);
      box.appendChild(hint);
      render();
      (document.body || document.documentElement).appendChild(box);
      if (window.matchMedia) {
        var mq = window.matchMedia('(max-width: 767px)');
        var onq = function () { if (box.parentNode) render(); };
        if (mq.addEventListener) mq.addEventListener('change', onq);
        else if (mq.addListener) mq.addListener(onq);
      }
      if (guide.shows < maxShows) {
        setTimeout(function () { painted = false; beginDelay(); }, coolSec * 1000);
      }
    }
    function arm(ms) {
      clearTimer();
      timer = setTimeout(function () {
        tickAt = 0;
        elapsed = 0;
        paint();
      }, ms);
      tickAt = Date.now();
    }
    function resume() {
      if (!visible() || painted) return;
      var left = delayMs - elapsed;
      if (left < 0) left = 0;
      arm(left);
    }
    function beginDelay() {
      elapsed = 0;
      tickAt = 0;
      if (visible()) resume();
    }
    document.addEventListener('visibilitychange', function () {
      if (!visible()) {
        if (tickAt) elapsed += Date.now() - tickAt;
        tickAt = 0;
        clearTimer();
        return;
      }
      if (!painted) resume();
    });
    var nowMs = Date.now();
    var nextMs = guide.next * 1000;
    if (nextMs > nowMs) setTimeout(beginDelay, nextMs - nowMs);
    else beginDelay();
  }
  function scheduleFromDecision(d) {
    if (!d || !d.handoffUrl || !d.handoffBanner || !d.handoffBanner.enabled) return;
    var b = d.handoffBanner;
    var maxAge = d.cookie && d.cookie.maxAge ? d.cookie.maxAge : 604800;
    schedule({
      url: d.handoffUrl,
      os: (d.inAppBrowser && d.inAppBrowser.os) || '',
      text: b.text || '',
      delaySeconds: b.delaySeconds,
      cooldownMinutes: b.cooldownMinutes,
      maxShows: b.maxShows,
      dismissEnds: !!b.dismissEnds,
      mobile: b.mobile,
      desktop: b.desktop,
      maxAge: maxAge
    });
  }
  function showPreset() {
    var p = window.__CLOAK_HANDOFF__;
    if (!p || !p.url || p.enabled === false) return;
    schedule({
      url: p.url,
      os: p.os || '',
      text: p.text || '',
      delaySeconds: p.delaySeconds,
      cooldownMinutes: p.cooldownMinutes,
      maxShows: p.maxShows,
      dismissEnds: !!p.dismissEnds,
      mobile: p.mobile,
      desktop: p.desktop,
      maxAge: 604800
    });
  }
  function fire(name, detail) {
    // bubbles 让 document 与 window 上的监听都能收到
    try { document.dispatchEvent(new CustomEvent(name, { detail: detail, bubbles: true })); } catch (e) {}
  }

  window.Cloak = window.Cloak || {};
  window.Cloak.getVisitorToken = window.Cloak.getVisitorToken || function () { return getCookie(cookieName); };
  window.Cloak.show = window.Cloak.show || show;
  window.Cloak.openInBrowser = openInBrowser;
  window.Cloak.showHandoff = scheduleFromDecision;
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
    if (document.body) { show(kind); scheduleFromDecision(d || {}); }
    else document.addEventListener('DOMContentLoaded', function () { show(kind); scheduleFromDecision(d || {}); });
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
