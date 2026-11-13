# 引导弹框技术方案

## 1. 文档信息

| 项目 | 内容 |
| --- | --- |
| 对应 PRD | `docs/v2/prd-handoff-prompt.md` v1.0 |
| 状态 | 已实现 |
| 改动面 | 站点配置、快照、判定返回、`cloak.js`、WordPress 插件插入的配置、商户端站点表单 |
| 不改 | 接力令牌签发与核销、模糊匹配、外链改写的匹配规则、判定结果本身 |

弹框只由 `backend-go/internal/transport/http/openapi/sdk/cloak.js` 绘制。网关 `internal/proxy/html.go` 不再为引导条拼一段内联 HTML。PHP 插件只下发配置和脚本地址，不画弹框。

## 2. 和现在实现的差异

| 现在 | 改后 |
| --- | --- |
| `cloak.js` 的 `banner()` 在拿到 `handoffUrl` 后立刻插入 `#ck-handoff` | 先累计前台可见时间，默认 10 秒 |
| 关闭写会话 Cookie `_ck_hb=1`，本次会话不再出现 | 写 `_ck_hg`，记录次数和下次允许时间 |
| `handoffBanner` 只有 `enabled` 和 `text` | 增加延迟、冷却、次数、两端文案 |
| 审核页插件也会塞 `__CLOAK_HANDOFF__` | 仅 `content` 为 `real` 或 `bypass` 时塞 |
| 后台一个预览弹窗、一句文案 | 移动端 / 电脑端两套文案、两个预览页签 |

`_ck_hb=1` 仍读取：存在则本次会话不弹，避免已关闭的访客在升级后立刻再看到。新代码不再写入 `_ck_hb`。

## 3. 配置

站点增加 JSON 列 `sites.handoff_prompt`。`handoff_banner`、`handoff_banner_text` 保留一个版本：旧文案在新字段说明为空时作为两端说明。商户保存新文案后，接口只写 JSON，并把 `handoff_banner_text` 写成移动端说明，避免旧读取路径变成空。

```json
{
  "delaySeconds": 10,
  "cooldownMinutes": 30,
  "maxShows": 3,
  "dismissEnds": false,
  "mobile":  { "title": "", "body": "", "open": "", "stay": "" },
  "desktop": { "title": "", "body": "", "open": "", "stay": "" }
}
```

校验与默认值按 PRD 第 11 节。超长截断，不返回 400。数字的回落规则：

| 字段 | 未传、null、空串 | 合法值 | 越界 |
| --- | --- | --- | --- |
| `delaySeconds` | 存 10 | 0–120，**0 原样保存** | 收进 0–120 |
| `cooldownMinutes` | 存 30 | 5–1440 | 收进 5–1440 |
| `maxShows` | 存 3 | 1–5 | 收进 1–5 |
| `dismissEnds` | 存 false | true / false | 当 false |

旧站点 `handoff_prompt` 为 NULL 时不回填数据库，读的时候按上表给出 10、30、3。这样升级不会改掉已有站点的行为，后台一打开也看得到初始值。商户保存其它站点字段时，若请求里没带这三项，保持列里的原值；列仍是 NULL 就继续当默认，不写成 0。

快照 `snapshot.Handoff` 增加同结构的 `Prompt`。`Banner` 仍是总开关。`BannerText` 只作迁移期回落。重建快照时把 JSON 放进 `Prompt`；列为空则 `delaySeconds=10`、`cooldownMinutes=30`、`maxShows=3`，`body` 用 `BannerText`。

管理接口沿用站点保存。`GET` 站点详情里的 `handoffPrompt` 永远带上这三个数字（库里没有就填 10、30、3）和展开后的文案，表单直接绑定，不再在前端写第二套兜底。`PUT` 按上表收齐后再入库。

## 4. 判定返回

`POST /api/open/v2/decision` 与 `POST /api/open/v2/js/capture` 在同时满足「总开关开、内置浏览器、`content` 为 `real` 或 `bypass`」时返回：

```json
{
  "handoffUrl": "https://shop.example/landing?code=ALICE01&_ck=...",
  "handoffBanner": {
    "enabled": true,
    "delaySeconds": 10,
    "cooldownMinutes": 30,
    "maxShows": 3,
    "dismissEnds": false,
    "mobile":  { "title": "在浏览器中打开", "body": "…", "open": "打开", "stay": "继续浏览" },
    "desktop": { "title": "用系统浏览器打开", "body": "…", "open": "打开", "stay": "稍后" }
  }
}
```

不满足时不要带 `handoffBanner.enabled=true`。审核结果不带 `handoffUrl`。文案在网关填好默认，SDK 不再内置第二套兜底长文案，只在字段缺失时用「打开 / 继续浏览」。

`inAppBrowser.os` 仍返回 `ios` 或 `android`，供跳转使用。端的选择不看 OS，看视口，见第 6 节。

## 5. JS SDK

`cloak.js` 用一个调度器替换现在的立即 `banner()`。`data-banner="preset"` 与判定返回走同一个调度器。`data-banner="off"` 不调度。

### 5.1 输入

- 判定模式：`apply()` 里，仅当 `content` 是 `real` 或 `bypass` 且 `handoffBanner.enabled` 时调用 `schedule(d)`。
- 预设模式：读 `window.__CLOAK_HANDOFF__`。插件保证只在放行且内置浏览器时写入。对象形状：

```js
{
  url: "https://shop.example/landing?_ck=...",
  os: "ios",
  enabled: true,
  delaySeconds: 10,
  cooldownMinutes: 30,
  maxShows: 3,
  dismissEnds: false,
  mobile: { title: "", body: "", open: "", stay: "" },
  desktop: { title: "", body: "", open: "", stay: "" }
}
```

`preset` 不再因为存在 `url` 就立刻画。缺的数字用 PRD 默认。

### 5.2 Cookie

名 `_ck_hg`。值 `v1.<shows>.<next>.<left>`，三段都是十进制整数。

| 段 | 含义 |
| --- | --- |
| `shows` | 已弹出次数 |
| `next` | 下次允许的 Unix 秒。0 表示不限制 |
| `left` | 1 表示已判定跳走 |

属性：`Path=/`、`SameSite=Lax`、HTTPS 加 `Secure`、`Max-Age` 取页面上访客 Cookie 的 `maxAge`，没有则 7 天，上限 30 天。解析失败当没有记录。

`_ck_hb=1` 或 `left=1` 或 `shows>=maxShows`：本页不调度。

### 5.3 计时

```text
可见秒 = 0
若 document.visibilityState === "visible" 则开始累加
visibilitychange 切到 hidden 时停表，回到 visible 续加
可见秒 >= delaySeconds 且 现在 >= next 且 页面上没有 #ck-handoff
    → show()
```

不使用 `setInterval` 每秒改 DOM。用 `setTimeout` 计算剩余毫秒；隐藏时 `clearTimeout`，显示时按剩余毫秒重设。

`show()` 成功插入节点后：

- `shows += 1`
- `next = now + cooldownMinutes * 60`
- 写 Cookie
- 若 `shows < maxShows`，再 `setTimeout(cooldown)`，到点且页面仍可见、仍没有弹框，则把可见秒清零，再等一个 `delaySeconds` 后 `show()`。人一直不换页也能走到第 2、3 次。

同一文档内 `show()` 用一个 `shown` 标记挡住并发。换页由新文档重新跑脚本，靠 Cookie 的 `next` 挡住冷却期。

前端路由：`cloak:decision` 再次触发时，若本次还没 `show()`，重新按新的 `handoffUrl` 调度；若已经 `show()` 过，本次文档不再因路由再计次。整页跳转自然是新文档。

### 5.4 关闭

关闭按钮和次按钮：

- 隐藏并移除 `#ck-handoff`
- 若 `dismissEnds`，把 `shows` 写成 `maxShows`
- `next` 保持 `show()` 时写过的冷却点，不缩短

不在关闭时再加次数。

### 5.5 打开

主按钮 `click` 使用 `openInBrowser(handoffUrl, os)`，与现在相同：iOS `x-safari-https://`，Android `intent://` 加 `BROWSABLE` 和 `S.browser_fallback_url`，其它系统直接打开 `handoffUrl`。必须在用户点击里赋值 `location`，不能自动跳。

点击后：

```text
2 秒内 visibilityState 变为 hidden，或 pagehide
    → left = 1，写 Cookie，移除弹框
仍为 visible
    → 弹框留着
    → os === ios 时若还没有提示行，追加「如果没有离开本页，请点右上角菜单，选择在 Safari 中打开。」
    → 不改 shows，不把 next 提前
```

## 6. 端与样式

`matchMedia("(max-width: 767px)")`。匹配用 `mobile`，否则用 `desktop`。监听 `change`，已存在的弹框只换文案和位置，不调用 `show()` 的计数。

样式继续写在节点上，不依赖客户站 CSS。`z-index` 用最大整数。根节点 `pointer-events:auto`，不创建遮罩层。移动端 `position:fixed; left:12px; right:12px; bottom: calc(12px + env(safe-area-inset-bottom))`。电脑端 `right:24px; bottom:24px; width:360px`。

标题、说明用 `textContent`。按钮不用客户提供的 HTML。

## 7. 谁插入脚本

| 入口 | 职责 |
| --- | --- |
| WordPress 插件 | 仅放行且内置浏览器且总开关开时，在 `</body>` 前写 `__CLOAK_HANDOFF__` 和 `sdk.js?data-banner=preset`。审核页不再写这段 |
| 网关托管 | 放行 HTML 不再注入引导条 DOM。在 `</body>` 前插入同一段配置和 `sdk.js`。外链改写仍只处理 `text/html` |
| 自建后端 | 判定响应里的 `handoffBanner` 交给已经引入的 `sdk.js`。文档示例改为等待 SDK，不自己画框 |

插件和网关插入的配置来自快照里已经填好默认值的 `Prompt`，加上本次 `handoffUrl` 和 `os`。

## 8. 后台预览

商户站点表单「跨浏览器接力」用详情接口返回的数字，不在前端另写一套初始值：

- 「出现延迟」数字框，单位秒，旁边是接口给的值。新建或未配置时是 10。最小值 0，最大值 120。
- 「再次提醒冷却」数字框，单位分钟。未配置时是 30。最小值 5，最大值 1440。
- 「最多提醒次数」数字框。未配置时是 3。最小值 1，最大值 5。
- 「关闭后不再提醒」开关。未配置时关。
- 原「提示文案」拆成「移动端」「电脑端」四字段。

清空数字框后点保存，请求里该字段为空，服务端写回 PRD 初始值（延迟写回 10，不把空当成 0）。用户明确把延迟改成 0 并保存，则存 0。
- 预览对话框两个页签。页签内假页面加对应宽度的弹框组件，数据来自表单当前值，不请求站点。
- 「打开」按钮 `@click.prevent`，用 `EleMessage` 提示线上会交给系统默认浏览器。

预览组件不读取 `_ck_hg`。

## 9. 测试

SDK 用无头页面或现有 js 测试夹具，不依赖真实社媒 App。

- 可见 10 秒前节点不存在；满 10 秒存在 `#ck-handoff`，且 `shows=1`、`next` 约为现在加 1800 秒。
- `visibilityState=hidden` 的 10 秒不计。
- `next` 未到时新文档不插入节点。
- `shows=3` 不再插入。
- `dismissEnds` 时关闭把 `shows` 写成 3。
- 点击打开后派发 `visibilitychange(hidden)`，Cookie `left=1`。不派发则弹框仍在，iOS 多一行提示。
- 宽度 390 用移动端说明，宽度 800 用电脑端说明；把移动端说明留空时显示电脑端说明。
- 审核内容的 `schedule` 不被调用。
- 快照：空 JSON 加旧 `BannerText` 时，返回的两端 `body` 都是旧文案。

PHP 插件用现有单测断言：审核决策不输出 `__CLOAK_HANDOFF__`，放行且内置浏览器时输出 `delaySeconds` 与两端文案。

## 10. 发布顺序

1. 先上网关和 SDK：旧插件只传 `text` 时，SDK 把它当作两端说明，延迟和冷却用默认。旧插件仍会在审核页插入脚本的窗口期内，SDK 若看到 `enabled` 但没有 `content` 标记，预设模式按插件写入与否执行。因此插件必须同一批改掉「审核页也插入」。
2. 再上商户端表单。旧字段继续能保存，避免表单先上、接口未识别新字段时把文案存丢。
3. 不要求客户重装插件即可使用默认 10 秒和默认重试；两端不同文案要等插件会把 JSON 写入 `__CLOAK_HANDOFF__`。托管网关不依赖插件。
