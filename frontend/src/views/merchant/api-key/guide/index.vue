<template>
  <ele-page>
    <ele-card :body-style="{ padding: '24px 28px' }">
      <div class="guide-header">
        <el-button text type="primary" @click="goGuide">
          <el-icon><ArrowLeft /></el-icon>
          返回接入指南
        </el-button>
        <h2 class="guide-title">判定接口对接文档（X-API-Key 方式）</h2>
        <p class="guide-desc">
          你的服务器在输出页面前调用判定接口，系统根据链接里的归因码、点击参数和访客
          Cookie 返回
          <code>page=real</code>（真实页）或
          <code>page=review</code>（审核页），由你决定输出哪套内容。地址栏保持访客打开的链接，不需要跳转。
        </p>
      </div>

      <el-alert
        type="success"
        :closable="false"
        show-icon
        class="guide-alert"
        title="推荐用「服务端 SDK + 深度代理」：判定走网关（只传一次 JSON），审核内容的回源与改写在你自己的服务器里完成，访客流量不经过网关。PHP 直接用插件（第 5 章）或 PHP SDK（第 7 章）；其他后端语言按本页判定接口契约自行实现。不想改代码、接受流量过网关的，用整站托管代理（第 6 章）。本章下面是最简单的 X-API-Key 直连方式。"
      />

      <div class="guide-section">
        <h3>0. 当前站点</h3>
        <el-form inline @submit.prevent="">
          <el-form-item label="站点">
            <el-select
              v-model="siteId"
              placeholder="选择站点后自动填入网关地址与密钥前缀"
              style="width: 320px"
            >
              <el-option
                v-for="s in sites"
                :key="s.siteId"
                :label="s.name"
                :value="s.siteId"
              />
            </el-select>
          </el-form-item>
        </el-form>
        <div class="kv">
          <span>网关地址</span><em class="mono">{{ baseUrl }}</em>
        </div>
        <div class="kv">
          <span>密钥前缀</span><em class="mono">{{ secretPrefix }}</em>
        </div>
      </div>

      <div class="guide-section">
        <h3>1. 获取服务端密钥</h3>
        <ul class="guide-list">
          <li
            >进入「站点管理」→ 打开站点 → 密钥区域，创建或重置<strong>服务端密钥</strong>。密钥以
            <code>sk_</code> 开头，只显示一次，请立即保存。</li
          >
          <li>密钥只能放在服务器配置或环境变量里，不能写进前端 JS 或页面源码。</li>
          <li
            >如果站点开启了密钥 IP 白名单，要把你的服务器出口 IP 加进去，否则返回
            <code>ip_not_allowed</code>。</li
          >
        </ul>
      </div>

      <div class="guide-section">
        <h3>2. 调用判定接口</h3>
        <p
          >每个前台页面请求，在生成 HTML 之前由服务器发起一次调用。请求头带
          <code>X-API-Key: sk_...</code>（也支持
          <code>Authorization: Bearer sk_...</code>）。</p
        >
        <div class="code-block">
          <div class="code-label">请求地址</div>
          <pre>POST {{ baseUrl }}/api/open/decision</pre>
        </div>
        <div class="code-block">
          <div class="code-label">请求头</div>
          <pre>Content-Type: application/json
X-API-Key: {{ secretPrefix }}…（完整 sk_ 密钥）</pre>
        </div>
        <div class="code-block">
          <div class="code-label">请求体（JSON）</div>
          <pre>
{
  "url": "https://{{ hostName }}/landing?code=ALICE01",   // 必填：访客正在打开的完整地址，归因码就在这里
  "ip": "访客 IP",                                      // 建议：Cloudflare 下取 CF-Connecting-IP
  "userAgent": "访客 User-Agent",
  "referer": "来源地址",
  "visitorToken": "访客 Cookie {{ cookieName }} 的值（没有就传空）"
}</pre
          >
        </div>
        <p class="guide-note"
          ><strong>url 不能省。</strong>没有访客地址接口会返回「缺少
          url」。也可以改传 <code>host</code>、<code>path</code>、<code>query</code>
          三个字段，效果相同。</p
        >
        <div class="code-block">
          <div class="code-label">cURL 示例</div>
          <pre>
curl -X POST "{{ baseUrl }}/api/open/decision" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: {{ secretPrefix }}…" \
  -d '{"url":"https://{{ hostName }}/landing?code=ALICE01","ip":"1.2.3.4","userAgent":"Mozilla/5.0"}'</pre
          >
        </div>
      </div>

      <div class="guide-section">
        <h3>3. 处理返回结果</h3>
        <el-table
          :data="resultRows"
          border
          style="width: 100%; max-width: 720px"
        >
          <el-table-column prop="field" label="字段" width="150" />
          <el-table-column prop="meaning" label="含义与用法" />
        </el-table>
        <div class="code-block" style="margin-top: 16px">
          <div class="code-label">响应示例（放行）</div>
          <pre>
{
  "code": 200,
  "msg": "ok",
  "data": {
    "page": "real",
    "action": "black",
    "content": "real",
    "attributed": true,
    "code": { "code": "ALICE01", "name": "Alice", "channel": "tiktok" },
    "visitorToken": "eyJ…",
    "cookie": { "name": "{{ cookieName }}", "value": "eyJ…", "maxAge": 2592000, "path": "/", "secure": true, "httpOnly": true, "sameSite": "Lax" },
    "handoffUrl": "https://{{ hostName }}/landing?code=ALICE01&_ck=…",
    "handoffBanner": { "enabled": true, "text": "在浏览器中打开，继续访问" }
  }
}</pre
          >
        </div>
        <ul class="guide-list">
          <li
            ><code>page=real</code>：输出真实页。顺手把
            <code>cookie</code> 写进访客浏览器，之后这个访客不带归因码再来，接口也会返回
            real。</li
          >
          <li><code>page=review</code>：输出审核页。地址不变，状态码 200。</li>
          <li
            ><code>handoffUrl</code> 有值：把这个地址交给页面上的 JS SDK 画引导弹框。弹框由 SDK 绘制，点击后打开系统浏览器，仍算同一个访客。引入位置见第 8 章。</li
          >
          <li
            >接口超时或返回非 200：按审核页处理，不要把真实页露出来。</li
          >
        </ul>
      </div>

      <div class="guide-section">
        <h3>4. 判定规则</h3>
        <ul class="guide-list">
          <li
            >链接带了<strong>有效的归因码</strong>（已登记、已启用、在生效时间内）→
            real。归因码在「归因码」菜单里登记。</li
          >
          <li>归因码过期、停用、未登记 → review。</li>
          <li
            >带有效<strong>点击参数</strong>（如 ttclid、fbclid，按「归因规则」配置）→
            real。</li
          >
          <li>没有参数但带有效 <code>visitorToken</code> → real（回访）。</li>
          <li
            >「归因规则」里可设置参数组合方式、点击参数复用限制、归因有效期等。</li
          >
        </ul>
      </div>

      <div class="guide-section guide-section--wordpress">
        <h3>5. WordPress 接入（插件，含深度代理）</h3>
        <p>
          推荐直接安装插件 <strong>cloak-attribution</strong>（v{{ pluginVersion }} 起内置<strong>深度代理</strong>），不用写代码。点下面按钮下载压缩包，到
          WordPress 后台「插件 → 安装插件 → 上传插件」安装启用，再填网关地址与服务端密钥即可。更完整的配置见
          <el-link type="primary" :underline="false" @click="goGuide"
            >接入指南 → WordPress</el-link
          >。下面是手写 PHP 的最小做法，适合主题里自定义；非 WordPress 的 PHP 程序用第 7 章的 PHP SDK。
        </p>
        <el-alert
          type="success"
          :closable="false"
          show-icon
          style="margin: 4px 0 12px"
          title="插件的「审核内容 → 深度代理整站」模式：审核页是另一个完整站点时，插件在本机把它整站代理过来，并把审核站域名改写成本站域名，站内点击继续停留在本域名、地址不变。流量全程在你的服务器，不经过网关。见下方 5.8。"
        />
        <el-button type="primary" plain @click="downloadPlugin">
          <el-icon style="margin-right: 4px"><Download /></el-icon>
          下载插件压缩包 v{{ pluginVersion }}
        </el-button>

        <h4>5.1 共用代码：调用判定接口</h4>
        <p
          >放进主题 <code>functions.php</code> 或自定义插件。密钥建议写进
          <code>wp-config.php</code>。</p
        >
        <div class="code-block">
          <div class="code-label">wp-config.php</div>
          <pre>define('CLOAK_API_BASE', '{{ baseUrl }}');
define('CLOAK_API_KEY', '{{ secretPrefix }}…');  // 完整 sk_ 密钥</pre>
        </div>
        <div class="code-block">
          <div class="code-label">共用代码（后面几种方案都要先放这段）</div>
          <pre>{{ sharedCode }}</pre>
        </div>

        <h4>5.2 方案一：审核页用平台模板，真实页继续当前页面（推荐）</h4>
        <p
          >不用在 WordPress 里另建页面。审核页用商户后台配置的平台模板，接口会在
          <code>renderHint.html</code> 里直接给出 HTML；放行时什么都不做，让
          WordPress 继续渲染访客打开的这篇页面。地址始终不变。</p
        >
        <div class="code-block">
          <div class="code-label">方案一</div>
          <pre>{{ templateCode }}</pre>
        </div>

        <h4>5.3 方案二：审核页是服务器上的 HTML 文件</h4>
        <p
          >把审核页做成独立 <code>review.html</code>，放在 Web 目录外更安全。判定为
          review 时读取文件直接输出，地址不变。</p
        >
        <div class="code-block">
          <div class="code-label">方案二</div>
          <pre>{{ htmlFileCode }}</pre>
        </div>

        <h4>5.4 方案三：审核页是 WordPress 站内页面</h4>
        <p
          >在后台新建一个「审核页」并记下页面 ID。判定为 review 时在<strong>当前地址</strong>渲染这个页面，不跳转。</p
        >
        <div class="code-block">
          <div class="code-label">方案三</div>
          <pre>{{ wpPageCode }}</pre>
        </div>

        <h4>5.5 兼容旧版：302 跳转</h4>
        <p
          >如果你的落地页必须是另一个固定网址，可以用跳转。注意浏览器地址会变、会闪一下，对回访和广告落地页不友好，优先用上面三种。<strong>想让那个外部网址整站代理、地址不变，请看下面第
          6 章的托管模式，不要用 302。</strong></p
        >
        <div class="code-block">
          <div class="code-label">跳转写法</div>
          <pre>{{ redirectCode }}</pre>
        </div>

        <h4>5.6 注意事项</h4>
        <ul class="guide-list">
          <li
            ><strong>全页缓存：</strong>WP Rocket、LiteSpeed、Nginx
            全页缓存会把第一个访客的页面发给所有人。审核 / 真实内容必须加
            <code>Cache-Control: no-store</code>，并把前台排除出缓存。</li
          >
          <li
            ><strong>后台与接口：</strong>示例代码已跳过
            <code>wp-admin</code>、Ajax、Cron、REST，后台和支付回调不受影响。</li
          >
          <li
            ><strong>真实 IP：</strong>套了 Cloudflare 时取
            <code>CF-Connecting-IP</code>，否则判定日志里看到的全是 CDN IP。</li
          >
          <li
            ><strong>接口失败：</strong>默认输出审核页。确认业务允许后再改成真实页。</li
          >
          <li
            ><strong>调试：</strong>「开发者中心 → API 调用日志」能看到每次调用的
            <code>page</code> 和耗时；「判定测试」可以不改代码直接模拟一次访问。</li
          >
        </ul>

        <h4>5.7 测试步骤</h4>
        <ol class="guide-steps">
          <li>无痕窗口打开首页：应看到审核页，地址不变，响应 200。</li>
          <li
            >打开 <code>https://{{ hostName }}/?code=已登记的归因码</code>：应看到真实页。</li
          >
          <li>同一窗口再打开不带参数的首页：仍是真实页（Cookie 生效）。</li>
          <li>到「API 调用日志」确认两条记录的 <code>page</code> 分别为 review 和 real。</li>
          <li>清一次站点缓存再重复第 1 步，确认没有被缓存。</li>
        </ol>

        <h4>5.8 深度代理整站（插件自带，推荐审核方式）</h4>
        <p
          >审核页是另一个<strong>完整站点</strong>（而不是单页模板）时用这个。插件在本机按访客当前路径回源审核站，把页面、脚本、样式、表单、<code>Location</code>、<code>Set-Cookie</code> 里的审核站域名改写成本站域名后在当前地址输出——<strong>地址栏不变、站内点击不跳走</strong>，访客流量全程在你的服务器，不经过网关。</p
        >
        <p>不用写代码，在插件设置里配置：</p>
        <ol class="guide-steps">
          <li>设置 → Cloak 归因 →「审核内容」→ 输出方式，选 <strong>深度代理整站（推荐）</strong>。</li>
          <li>第一个框填审核站地址，如 <code>https://review.example.com</code>。</li>
          <li
            >源站用 IP 部署、证书是另一个域名时，第二个框填<strong>回源 Host</strong>（证书 / 虚拟主机名），插件回源握手时带上它；普通情况留空。</li
          >
          <li>保存。无痕打开本站应看到审核站整站内容，地址仍是本站域名，点站内链接也不跳到审核站。</li>
        </ol>
        <ul class="guide-list">
          <li
            >与「服务端拉取外部地址」的区别：后者只把目标页抓回来原样输出，页内链接仍指向审核站；<strong>深度代理会把审核站域名整体改写成本站域名</strong>，实现整站挂在本域名下。</li
          >
          <li
            >确保你的 WordPress 服务器能访问审核站；审核站若有防火墙 / IP 白名单，放行你的服务器出口 IP。</li
          >
          <li
            >只改写文本类型（HTML / CSS / JS / JSON / XML / SVG），图片、字体、二进制原样透传；<code>cdn.review.com</code> 这类子域名、第三方 CDN 保持原样。</li
          >
        </ul>
      </div>

      <div class="guide-section">
        <h3>6. 整站代理（网关托管）：广告投客户主域名，审核站深度代理</h3>
        <el-alert
          type="warning"
          :closable="false"
          show-icon
          class="guide-alert"
          title="广告投放 example.com，用 CNAME 指向网关入口。网关以后换服务器，客户不用再改解析。origin.example.com 只给网关回源，指向原来的网站服务器。review.com 是审核站，解析不要改。"
        />
        <p
          >本章是<strong>网关托管</strong>的整站代理：不写代码，广告继续投放客户主域名，访客流量经过网关，由网关按有没有归因码取真实网站或审核站并完成深度代理改写。审核站整站出现在客户主域名下，地址栏和站内链接都不变。</p
        >
        <el-alert
          type="info"
          :closable="false"
          show-icon
          style="margin: 0 0 12px"
          title="与 SDK / 插件深度代理的区别：本章的回源与改写在我们的网关上执行，访客流量过网关；第 5 章插件、第 7 章 PHP SDK 的深度代理在你自己的服务器上执行，访客流量不经过网关。想减轻网关带宽、流量留在本地，用 SDK / 插件；完全不想碰服务器代码，用本章。"
        />
        <p
          >下面用三个示例名字说明。接入时换成你自己的域名。网关入口是
          <em class="mono">{{ entryHost || '网关入口域名' }}</em>。</p
        >

        <h4>6.1 访客实际看到什么</h4>
        <ul class="guide-list">
          <li
            >投放链接始终是 <code>https://example.com/?code=归因码</code>。带有效归因码时，内容是客户原来的网站。</li
          >
          <li
            >没有归因码时，网关去取审核站 <code>https://review.com</code>，再把页面里的链接、图片、脚本、表单、样式和接口地址里的审核站域名改成
            <code>example.com</code>。跳转头 <code>Location</code> 和
            <code>Set-Cookie</code> 的域名一并改写。</li
          >
          <li
            >之后在审核内容里点分类、商品，地址仍是
            <code>example.com</code>，内容仍来自审核站。带归因码的访客点开后续页面，仍是真实网站。</li
          >
          <li>不需要 WordPress 插件，也不要在源站里再调用一次判定接口。</li>
        </ul>

        <h4>6.2 三个地址分别指向哪里</h4>
        <table class="guide-dns">
          <thead>
            <tr>
              <th>地址</th>
              <th>谁来访问</th>
              <th>解析指向</th>
              <th>后台怎么填</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td><code>example.com</code><br />客户主域名</td>
              <td>广告和所有访客</td>
              <td
                >网关。用 <strong>CNAME</strong>，指向
                <code>{{ entryHost || '网关入口域名' }}</code>
                。同一条主机名上不要再留 A 记录。网关换服务器后，这条 CNAME 不用改。</td
              >
              <td>域名管理添加为<strong>托管域名</strong>，校验到<strong>已绑定</strong></td>
            </tr>
            <tr>
              <td><code>origin.example.com</code><br />只给网关用的源站地址</td>
              <td>只有网关</td>
              <td
                ><strong>客户网站原来的服务器 IP</strong>，也就是主域名改到网关之前的那个
                IP。不要指向网关。<strong
                  >这台源站的防火墙要放行网关回源 IP
                  <code>{{ entryIP || '网关回源 IP' }}</code></strong
                >，否则网关取不回真实页。</td
              >
              <td
                >真实源站填 <code>https://origin.example.com</code>。回源 Host 填
                <code>example.com</code></td
              >
            </tr>
            <tr>
              <td><code>review.com</code><br />审核站</td>
              <td>只有网关</td>
              <td>保持审核站自己的服务器，解析不要改</td>
              <td
                >审核源站填 <code>https://review.com</code>。回源 Host 留空。审核内容选<strong>审核源站</strong></td
              >
            </tr>
          </tbody>
        </table>
        <p class="guide-note"
          >真实网站如果本来就是另一个一级域名，访客和广告从不打开它，真实源站可以直接填那个域名，解析继续指向真实站服务器。回源 Host 留空。广告域名和真实网站是同一个一级域名时，再准备只给网关用的
          <code>origin.example.com</code>，A 记录指向原来的网站服务器。</p
        >

        <h4>6.3 源站防火墙放行网关回源 IP（务必配置）</h4>
        <el-alert
          type="error"
          :closable="false"
          show-icon
          class="guide-alert"
          :title="`真实网站的源服务器必须把网关回源 IP ${
            entryIP || '（见上方「网关入口」对应的 IP）'
          } 加入防火墙永久白名单。漏了这一步，带归因码的访客会退回审核站。`"
        />
        <p
          >网关取真实页时，是用固定的出口 IP
          <em class="mono">{{ entryIP || '网关回源 IP' }}</em>
          去连 <code>origin.example.com</code>。源站上的防火墙或安全软件会把这种「同一个
          IP 频繁、规律地回源」判成攻击或爬虫，自动把网关 IP 封掉——表现是网关一发起
          TLS 握手就被源站重置连接（<code>connection reset by peer</code>）。</p
        >
        <ul class="guide-list">
          <li
            ><strong>要加永久白名单，不是解一次封。</strong>只解封一次，几分钟后会被安全软件再次自动封禁。</li
          >
          <li
            ><strong>加在真正拦截的那一层。</strong>常见位置：主机防火墙
            <code>CSF</code>（<code>csf -a 网关IP</code>，并从
            <code>csf.deny</code> 里删掉）、<code>Imunify360</code>、宝塔防火墙、WordPress
            的 <code>Wordfence</code>、以及源站前置的
            <code>Cloudflare</code> 回源白名单。源站若是共享主机、你没权限，就开工单让主机商把网关
            IP 永久加白。</li
          >
          <li
            ><strong>怎么确认已放行：</strong>站点配置里真实源站点「检测」显示可达；判定日志里带归因码的请求
            <code>origin_kind=real</code> 且
            <code>fallback=0</code>。若显示 <code>origin_kind=review</code>、<code>fallback=1</code>
            或源站健康为「不可达 / connection reset by peer」，说明还没放行。</li
          >
        </ul>

        <h4>6.4 这几项配反了会失败</h4>
        <ul class="guide-list">
          <li
            ><strong>源站防火墙没放行网关回源 IP
            <code>{{ entryIP || '网关回源 IP' }}</code>。</strong>网关回源被源站重置，真实页取不回来，带归因码的访客也退回审核站。见上一节 6.3。</li
          >
          <li
            ><strong>真实源站填了 <code>https://example.com</code>。</strong>主域名已经指向网关，网关再向它回源会绕回自己，页面返回
            508。</li
          >
          <li
            ><strong><code>origin.example.com</code> 的 A 记录指向了网关。</strong>真实页同样绕回网关，打不开。它必须和原来的网站服务器是同一个 IP。</li
          >
          <li
            ><strong>主域名仍然指向原来的网站服务器。</strong>广告不经过网关，没有归因码时不会换成审核站。</li
          >
          <li
            ><strong>把审核站域名也改指向网关。</strong>审核内容取不回来。审核站保持原解析。</li
          >
          <li
            ><strong>回源 Host 留空，源站地址却是一个没有证书的新域名。</strong>回源 Host 填
            <code>example.com</code> 后，网关按客户主域名上已有的证书握手，不用再配一张新证书。</li
          >
          <li
            ><strong>客户主域名没有 CNAME 到网关入口。</strong>应把
            <code>example.com</code> 的 CNAME 指到
            <code>{{ entryHost || '网关入口域名' }}</code>。这条名字上如果还留着旧的 A
            记录，会和 CNAME 冲突，解析不会指向网关。</li
          >
        </ul>

        <h4>6.5 后台填写</h4>
        <ol class="guide-steps">
          <li>站点管理 → 打开站点。接入方式选 <strong>托管接入</strong> 或 <strong>混合接入</strong>，保存设置。</li>
          <li>无归因时选 <strong>返回审核内容</strong>。Cookie 域留空。</li>
          <li
            >域名里添加 <code>example.com</code>，类型选托管域名。把该域名的 CNAME 指到
            <code>{{ entryHost || '网关入口域名' }}</code>，去掉原来的 A 记录，再点校验，状态变为已绑定。</li
          >
          <li
            >真实源站填 <code>https://原服务器IP</code>，或填解析到原服务器的 <code>https://origin.example.com</code>。回源 Host 填
            <code>example.com</code>。网关用这张已有证书握手，不用再给源站域名单独配证书。<strong>先确认源站防火墙已放行网关回源
            IP <code>{{ entryIP || '网关回源 IP' }}</code>（见 6.3）</strong>，再保存点检测，状态应为可达。</li
          >
          <li
            >审核源站填 <code>https://review.com</code>，回源 Host 留空，保存后点检测。</li
          >
          <li>审核内容选 <strong>审核源站</strong>，保存审核内容。这里不要选平台模板，否则看到的是模板而不是审核站整站。</li>
        </ol>

        <h4>6.6 建议的操作顺序</h4>
        <ol class="guide-steps">
          <li>先在客户原来的服务器上添加 <code>origin.example.com</code>，A 记录指向当前网站 IP，并装好包含这个名字的证书。此时主域名先不要改。</li>
          <li
            >在客户源站的防火墙 / 安全软件里，把网关回源 IP
            <code>{{ entryIP || '网关回源 IP' }}</code> 加入<strong>永久白名单</strong>（见
            6.3），避免回源被源站自动封禁。</li
          >
          <li>在后台填好真实源站并检测通过。</li>
          <li>再把 <code>example.com</code> 改成 CNAME，指向 <code>{{ entryHost || '网关入口域名' }}</code>，并删掉原来的 A 记录。回到后台校验到已绑定。</li>
          <li>最后填审核源站并保存审核内容。</li>
        </ol>
        <p class="guide-note"
          >先打通 <code>origin.example.com</code>，再切主域名。主域名一切到网关，访客就不再直接碰到原来的网站，只能由网关经 <code>origin.example.com</code> 取真实页。</p>
        >

        <h4>6.7 深度代理会改写的内容</h4>
        <ul class="guide-list">
          <li>审核站 HTML 里的链接、图片、脚本、样式和表单。</li>
          <li>页面内脚本，以及单独返回的 JS、JSON、XML 里写死的审核站域名，包括 <code>https:\/\/review.com</code> 这种转义写法。</li>
          <li>CSS 里的 <code>url(...)</code>，以及 301/302 的 <code>Location</code>。</li>
          <li
            >审核站以外的其他域名和第三方 CDN 保持原样。脚本若把域名拆开再拼接，这种地址改不到。</li
          >
        </ul>

        <h4>6.8 验收</h4>
        <ol class="guide-steps">
          <li>无痕窗口打开 <code>https://example.com/</code>：地址栏是客户主域名，页面内容是审核站。再点站内链接，地址仍停在 <code>example.com</code>，不会跳到审核站域名。</li>
          <li>无痕窗口打开 <code>https://example.com/?code=已登记的归因码</code>：内容换成客户原来的网站，地址栏仍是主域名。</li>
          <li>同一窗口再打开不带参数的页面：仍是真实网站（归因 Cookie 还在）。换一个无痕窗口，又回到审核站内容。</li>
        </ol>
        <el-alert
          type="info"
          :closable="false"
          show-icon
          style="margin-top: 12px"
          title="本章和上面第 5 章的插件 / PHP 方案二选一。主域名已经指向网关时，不要再在 WordPress 里拉一次外部地址，插件不会把审核站链接改写成客户主域名。"
        />
      </div>

      <div class="guide-section guide-section--wordpress">
        <h3>7. PHP SDK（自研 PHP，推荐：深度代理在你的服务器）</h3>
        <el-alert
          type="success"
          :closable="false"
          show-icon
          class="guide-alert"
          title="适合非 WordPress 的 PHP 程序。SDK 在网站入口判定并分流：放行渲染你的真实内容；审核时在本机深度代理审核站——回源、把审核站域名改写成你的域名、再原地输出。判定走网关，访客页面与资源流量全程在你的服务器。"
        />

        <h4>7.1 安装</h4>
        <div class="code-block">
          <div class="code-label">Composer</div>
          <pre>composer require h5cloak/cloak-sdk-php</pre>
        </div>
        <p class="guide-note">也可直接引入 <code>sdk/php/src</code> 下的类（PSR-4，命名空间 <code>Cloak\Sdk\</code>）。</p>

        <h4>7.2 一行接入</h4>
        <p>放在网站入口（front controller）最前面。密钥建议用环境变量，不要写进代码库。</p>
        <div class="code-block">
          <div class="code-label">入口</div>
          <pre>{{ sdkPhpCode }}</pre>
        </div>

        <h4>7.3 手动分流（需要自己控制时）</h4>
        <div class="code-block">
          <div class="code-label">手动用法</div>
          <pre>{{ sdkPhpManualCode }}</pre>
        </div>

        <h4>7.4 常用配置</h4>
        <el-table :data="sdkConfigRows" border style="width: 100%; max-width: 720px">
          <el-table-column prop="key" label="配置项" width="160" />
          <el-table-column prop="meaning" label="说明" />
        </el-table>
        <p class="guide-note">默认用 ext-curl 回源。要换成框架自己的 HTTP 客户端，实现 <code>Cloak\Sdk\HttpTransport</code> 后注入：<code>new \Cloak\Sdk\Client($cfg, $myTransport)</code>。</p>

        <h4>7.5 注意事项</h4>
        <ul class="guide-list">
          <li><code>sk_</code> 只放环境变量 / 服务器配置，不要写进任何前端输出。</li>
          <li>SDK 的深度代理只代理你配置的 <code>reviewOrigin</code>，访客无法通过 path 控制回源目标（防 SSRF）。</li>
          <li>确保你的 PHP 服务器能访问审核站；审核站有防火墙时放行你的服务器出口 IP。</li>
          <li>站点若开了密钥 IP 白名单，把<strong>你的服务器出口 IP</strong> 加进去，否则判定返回 <code>ip_not_allowed</code>。</li>
          <li>只改写文本类型（HTML / CSS / JS / JSON / XML / SVG），图片、字体、二进制原样透传；<code>cdn.review.com</code> 这类子域名与第三方 CDN 保持原样。</li>
        </ul>

        <h4>7.6 其他后端语言</h4>
        <p
          >Node / Java / Go / Python 等暂无官方 SDK（规划中），按本页第 2–4 章的判定接口契约自行实现同样的流程：入口判定 → 放行渲染真实内容 → 审核在本机回源审核站并按深度代理规则改写域名后原地输出。可参考 <code>sdk/go</code> 的实现与 <code>sdk/testdata</code> 的黄金用例集对拍。引导弹框仍用下面第 8 章的 JS SDK，插在输出 HTML 的 <code>&lt;/body&gt;</code> 前。</p
        >

        <h3>8. JS SDK 放在哪</h3>
        <p
          >JS SDK 是客户站点页面里的一段脚本，地址是 <code>{{ baseUrl }}/_cloak/sdk.js</code>。引导弹框由它在浏览器里画出来，服务端不输出弹框 HTML。每个要生效的 HTML 页只引一次，放在 <code>&lt;/body&gt;</code> 之前。</p
        >

        <h4>8.1 WordPress 插件不用手写</h4>
        <p
          >插件 v{{ pluginVersion }} 起，访客在 Facebook、微信、抖音等内置浏览器里时，会自动把 SDK 插到页面 <code>&lt;/body&gt;</code> 前。主题里不要再贴一遍。</p
        >
        <ul class="guide-list">
          <li><code>wp-config.php</code> 里要有 <code>CLOAK_API_BASE</code>，脚本从这里加载。</li>
          <li>插件设置勾选「内置浏览器引导条」。</li>
          <li>站点配置 → 跨浏览器接力，打开「打开提示条」。</li>
        </ul>

        <h4>8.2 自研页面：只画弹框</h4>
        <p
          >PHP SDK 或其它后端已经在服务器判定过时，不要再带 <code>data-key</code> 请求一次判定。在你输出的 HTML（含深度代理后的审核页）<code>&lt;/body&gt;</code> 前插入：</p
        >
        <div class="code-block">
          <pre>{{ jsSdkPresetCode }}</pre>
        </div>
        <ul class="guide-list">
          <li><code>url</code>：已放行且接口返回了 <code>handoffUrl</code> 时用它，否则用当前页完整地址。</li>
          <li><code>os</code>：接口 <code>inAppBrowser.os</code>，<code>ios</code> 或 <code>android</code>。</li>
          <li><code>data-banner="preset"</code>：只画弹框并处理点击，不再调用判定接口。</li>
          <li>Android 点「打开」交给系统默认浏览器；iPhone 尝试交给 Safari。</li>
        </ul>

        <h4>8.3 静态页：在浏览器里判定</h4>
        <p>没有服务端、两套内容可以同时出现在源码里时，同样放在 <code>&lt;/body&gt;</code> 前，公钥用 <code>pk_</code>：</p>
        <div class="code-block">
          <pre>{{ jsSdkCode }}</pre>
        </div>
        <p class="guide-note"
          >打开页面的域名必须已绑定到该站点。查看源代码能看到两套内容；要隐藏真实 HTML，用第 5 章或第 7 章。<code>data-banner="off"</code> 关闭引导弹框。</p
        >

      </div>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { ref, computed, watch, onMounted } from 'vue';
  import { useRouter } from 'vue-router';
  import { ArrowLeft, Download } from '@element-plus/icons-vue';
  import { useCurrentSite } from '@/utils/use-site';
  import { getSite, getSiteGuide } from '@/api/merchant/site';

  defineOptions({ name: 'MerchantApiKeyGuide' });

  const router = useRouter();
  const { sites, siteId, load: loadSites } = useCurrentSite();
  const site = ref(null);
  const entryHost = ref('');
  const entryIP = ref('');

  const baseUrl = computed(() =>
    entryHost.value ? `https://${entryHost.value}` : 'https://<网关地址>'
  );
  const secretPrefix = computed(
    () => site.value?.secretPrefix || 'sk_xxxxxxxxx'
  );
  const cookieName = computed(() => site.value?.cookieName || '_ck_vid');
  const hostName = computed(() => {
    const d = site.value?.domains?.find?.((x) => x.status === 'bound');
    return d?.domain || 'example.com';
  });

  const loadSite = () => {
    if (!siteId.value) {
      site.value = null;
      return;
    }
    getSite(siteId.value)
      .then((d) => (site.value = d))
      .catch(() => {});
  };

  const goGuide = () => router.push('/merchant/api-guide');

  const pluginVersion = '2.3.7';
  const downloadPlugin = () => {
    const a = document.createElement('a');
    a.href = `${import.meta.env.BASE_URL}downloads/cloak-attribution.zip`;
    a.download = `cloak-attribution-${pluginVersion}.zip`;
    document.body.appendChild(a);
    a.click();
    a.remove();
  };

  const resultRows = [
    { field: 'page', meaning: 'real = 输出真实页；review = 输出审核页。只看这个字段就够了。' },
    { field: 'action', meaning: '与 page 对应的旧字段：black = 真实，white = 审核。' },
    { field: 'content', meaning: '更细的结果：real / bypass 放行，review 审核，block 403，not_found 404。' },
    { field: 'cookie', meaning: '放行时返回。按 name / value / maxAge 写入访客 Cookie，回访免带码。' },
    { field: 'visitorToken', meaning: '同 cookie.value。下次调用时原样放进请求体的 visitorToken。' },
    { field: 'code', meaning: '命中的归因码（码值、名称、渠道），可用于自己的埋点。' },
    { field: 'renderHint', meaning: '审核时返回。type=html 时 html 字段就是平台模板，可直接输出。' },
    { field: 'handoffUrl', meaning: '内置浏览器放行时返回。给访客一个「在浏览器中打开」的入口。' }
  ];

  const sharedCode = computed(
    () => `<?php
// 共用：取访客真实 IP
function cloak_client_ip() {
    foreach (['HTTP_CF_CONNECTING_IP', 'HTTP_X_REAL_IP', 'HTTP_X_FORWARDED_FOR', 'REMOTE_ADDR'] as $k) {
        if (empty($_SERVER[$k])) continue;
        $ip = trim(explode(',', $_SERVER[$k])[0]);
        if (filter_var($ip, FILTER_VALIDATE_IP)) return $ip;
    }
    return '';
}

// 共用：调用判定接口，返回 data 数组；失败返回 null
function cloak_decide() {
    $scheme = is_ssl() ? 'https' : 'http';
    $url = $scheme . '://' . ($_SERVER['HTTP_HOST'] ?? '') . ($_SERVER['REQUEST_URI'] ?? '/');
    $cookie = '${cookieName.value}';
    $body = wp_json_encode([
        'url'          => $url,                                   // 必填
        'ip'           => cloak_client_ip(),
        'userAgent'    => $_SERVER['HTTP_USER_AGENT'] ?? '',
        'referer'      => $_SERVER['HTTP_REFERER'] ?? '',
        'visitorToken' => $_COOKIE[$cookie] ?? '',
    ], JSON_UNESCAPED_SLASHES);

    $res = wp_remote_post(CLOAK_API_BASE . '/api/open/decision', [
        'timeout' => 3,
        'headers' => ['Content-Type' => 'application/json', 'X-API-Key' => CLOAK_API_KEY],
        'body'    => $body,
    ]);
    if (is_wp_error($res) || wp_remote_retrieve_response_code($res) !== 200) return null;
    $json = json_decode(wp_remote_retrieve_body($res), true);
    return $json['data'] ?? null;
}

// 共用：按接口返回写访客 Cookie
function cloak_set_cookie($data) {
    if (empty($data['cookie']['value']) || headers_sent()) return;
    $c = $data['cookie'];
    setcookie($c['name'], $c['value'], [
        'expires'  => time() + (int) ($c['maxAge'] ?? 2592000),
        'path'     => '/',
        'secure'   => is_ssl(),
        'httponly' => true,
        'samesite' => 'Lax',
    ]);
}

// 共用：哪些请求不判定
function cloak_should_skip() {
    return is_admin() || wp_doing_ajax() || wp_doing_cron() || is_feed() || is_robots()
        || (defined('REST_REQUEST') && REST_REQUEST);
}

// 共用：审核 / 真实内容都不能被全页缓存
function cloak_no_cache_headers($page) {
    nocache_headers();
    header('Cache-Control: no-store');
    header('X-Cloak-Content: ' . $page);
    if ($page === 'review') header('X-Robots-Tag: noindex, nofollow');
}`
  );

  const templateCode = `<?php
add_action('template_redirect', function () {
    if (cloak_should_skip()) return;

    $data = cloak_decide();
    $page = $data['page'] ?? 'review';        // 接口失败按审核页
    if ($data) cloak_set_cookie($data);

    if ($page === 'real') {
        cloak_no_cache_headers('real');
        return;                                // 继续渲染访客当前打开的页面
    }

    cloak_no_cache_headers('review');
    status_header(200);
    header('Content-Type: text/html; charset=utf-8');
    $hint = $data['renderHint'] ?? [];
    if (($hint['type'] ?? '') === 'html' && !empty($hint['html'])) {
        echo $hint['html'];                    // 商户后台配置的平台模板
    } else {
        echo '<!doctype html><html><head><meta charset="utf-8"><title>' . esc_html(get_bloginfo('name'))
           . '</title></head><body><h1>' . esc_html(get_bloginfo('name')) . '</h1></body></html>';
    }
    exit;
}, 1);`;

  const htmlFileCode = `<?php
// 放在 Web 目录外，只有 PHP 可读
define('CLOAK_REVIEW_HTML', dirname(ABSPATH) . '/cloak-pages/review.html');

add_action('template_redirect', function () {
    if (cloak_should_skip()) return;

    $data = cloak_decide();
    $page = $data['page'] ?? 'review';
    if ($data) cloak_set_cookie($data);

    if ($page === 'real') {
        cloak_no_cache_headers('real');
        return;
    }
    if (!is_readable(CLOAK_REVIEW_HTML)) return;   // 文件缺失时不白屏

    cloak_no_cache_headers('review');
    status_header(200);
    header('Content-Type: text/html; charset=utf-8');
    readfile(CLOAK_REVIEW_HTML);
    exit;
}, 1);`;

  const wpPageCode = `<?php
define('CLOAK_REVIEW_PAGE_ID', 123);   // 后台新建的审核页 ID

add_action('template_redirect', function () {
    if (cloak_should_skip()) return;
    if (is_page(CLOAK_REVIEW_PAGE_ID)) return;    // 审核页本身不再判定

    $data = cloak_decide();
    $page = $data['page'] ?? 'review';
    if ($data) cloak_set_cookie($data);

    if ($page === 'real') {
        cloak_no_cache_headers('real');
        return;
    }

    // 在当前地址渲染审核页，不跳转
    global $wp_query, $post;
    $post = get_post(CLOAK_REVIEW_PAGE_ID);
    $wp_query = new WP_Query(['page_id' => CLOAK_REVIEW_PAGE_ID, 'post_type' => 'page']);
    $wp_query->is_404 = false;
    $wp_query->is_page = $wp_query->is_singular = true;
    setup_postdata($post);
    cloak_no_cache_headers('review');
    status_header(200);
    $tpl = get_page_template();
    if ($tpl && file_exists($tpl)) { include $tpl; } else { get_header(); echo apply_filters('the_content', $post->post_content); get_footer(); }
    exit;
}, 1);`;

  const redirectCode = `<?php
define('CLOAK_REVIEW_URL', 'https://review.com/landing');   // 必须 https
define('CLOAK_REAL_PAGE_ID', 456);

add_action('template_redirect', function () {
    if (cloak_should_skip()) return;

    $data = cloak_decide();
    $page = $data['page'] ?? 'review';
    if ($data) cloak_set_cookie($data);          // 先写 Cookie 再跳，跳过去仍能识别

    if ($page === 'review') {
        if (wp_parse_url(CLOAK_REVIEW_URL, PHP_URL_SCHEME) !== 'https') return;
        wp_redirect(CLOAK_REVIEW_URL, 302);
        exit;
    }
    if (is_page(CLOAK_REAL_PAGE_ID)) return;
    wp_safe_redirect(get_permalink(CLOAK_REAL_PAGE_ID), 302);
    exit;
}, 1);`;

  const sdkPhpCode = computed(
    () => `<?php
require 'vendor/autoload.php';

$client = new \\Cloak\\Sdk\\Client([
    'apiBase'      => '${baseUrl.value}',
    'secretKey'    => getenv('CLOAK_SECRET'),          // sk_...，只放服务器
    'reviewOrigin' => 'https://review.example.com',    // 审核站地址
    // 'reviewHost' => 'review.example.com',           // 源站用 IP 部署时填回源 Host
]);

$client->handle();
// 放行：handle() 直接 return，下面继续渲染你的真实页面
// 审核 / 拦截：handle() 已输出并 exit`
  );

  const sdkPhpManualCode = `<?php
$d = $client->decide();              // 返回 data 数组，失败返回 null
if ($d && in_array($d['content'] ?? '', ['real', 'bypass'], true)) {
    // 放行：渲染真实内容（handle 会自动写 Cookie；手动时按 $d['cookie'] 自行写）
} else {
    $client->proxyReview();          // 在本机深度代理审核站并输出
    exit;
}`;

  const jsSdkPresetCode = computed(
    () => `<script>
  window.__CLOAK_HANDOFF__ = {
    url: "https://${hostName.value}/当前路径",
    os: "android",
    text: "在系统浏览器中打开",
    enabled: true
  };
<\/script>
<script src="${baseUrl.value}/_cloak/sdk.js" data-banner="preset" async><\/script>`
  );

  const jsSdkCode = computed(
    () => `<script src="${baseUrl.value}/_cloak/sdk.js" data-key="pk_你的公钥" async><\/script>

<div data-cloak="review" hidden>…审核内容…</div>
<div data-cloak="real" hidden>…真实内容…</div>`
  );

  const sdkConfigRows = [
    { key: 'apiBase', meaning: '网关地址（必填）' },
    { key: 'secretKey', meaning: '服务端密钥 sk_...（必填，只放服务器）' },
    { key: 'reviewOrigin', meaning: '审核站地址（必填）' },
    { key: 'visitorHost', meaning: '对外域名，留空用请求 Host' },
    { key: 'reviewHost', meaning: '回源 Host，源站用 IP 部署时填，留空用 reviewOrigin 主机' },
    { key: 'cookieName', meaning: '访客 Cookie 名，默认 _ck_vid' },
    { key: 'timeout', meaning: '判定 / 回源超时（秒），默认 3' },
    { key: 'failOpen', meaning: '判定失败是否放行真实内容，默认 false（按审核兜底）' },
    { key: 'skipPaths / skipSuffixes', meaning: '跳过判定的路径 / 后缀，默认含后台、登录、静态资源' },
    { key: 'clientIpHeaders', meaning: '取真实 IP 的头，默认 CF-Connecting-IP 等' }
  ];

  watch(siteId, loadSite);
  onMounted(() => {
    getSiteGuide()
      .then((d) => {
        entryHost.value = d.entryHost || '';
        entryIP.value = d.entryIP || '';
      })
      .catch(() => {});
    loadSites().then(loadSite);
  });
</script>

<style scoped>
  .guide-header {
    margin-bottom: 20px;
  }

  .guide-title {
    margin: 12px 0 8px;
    font-size: 22px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .guide-desc {
    margin: 0;
    color: var(--el-text-color-secondary);
    line-height: 1.6;
  }

  .guide-alert {
    margin-bottom: 24px;
  }

  .guide-section {
    margin-bottom: 28px;
  }

  .guide-section h3 {
    margin: 0 0 10px;
    font-size: 16px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .guide-section h4 {
    margin: 20px 0 8px;
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .guide-note {
    margin: 16px 0 8px;
    padding: 10px 14px;
    font-size: 13px;
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-lighter);
    border-radius: 6px;
    border-left: 3px solid var(--el-color-primary);
  }

  .guide-section--wordpress strong {
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .guide-steps {
    margin: 8px 0 0;
    padding-left: 20px;
    color: var(--el-text-color-regular);
    line-height: 1.9;
  }

  .guide-section p {
    margin: 0 0 12px;
    color: var(--el-text-color-regular);
    line-height: 1.7;
  }

  .guide-list {
    margin: 12px 0 0;
    padding-left: 20px;
    color: var(--el-text-color-regular);
    line-height: 1.8;
  }

  .guide-dns {
    width: 100%;
    margin: 8px 0 4px;
    border-collapse: collapse;
    font-size: 13px;
    line-height: 1.6;
    color: var(--el-text-color-regular);
  }

  .guide-dns th,
  .guide-dns td {
    border: 1px solid var(--el-border-color-lighter);
    padding: 8px 10px;
    text-align: left;
    vertical-align: top;
  }

  .guide-dns th {
    background: var(--el-fill-color-light);
    color: var(--el-text-color-primary);
    font-weight: 600;
  }

  .kv {
    display: flex;
    gap: 12px;
    font-size: 13px;
    line-height: 1.9;
    color: var(--el-text-color-regular);
  }

  .kv span {
    width: 72px;
    color: var(--el-text-color-secondary);
  }

  .kv em {
    font-style: normal;
    color: var(--el-text-color-primary);
  }

  .mono {
    font-family: Consolas, Menlo, monospace;
  }

  .code-block {
    margin-top: 12px;
    background: var(--el-fill-color-light);
    border-radius: 8px;
    overflow: hidden;
  }

  .code-label {
    padding: 8px 14px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .code-block pre {
    margin: 0;
    padding: 14px;
    font-size: 13px;
    line-height: 1.6;
    overflow-x: auto;
    white-space: pre-wrap;
    word-break: break-all;
    color: var(--el-text-color-primary);
  }

  code {
    padding: 2px 6px;
    font-size: 13px;
    background: var(--el-fill-color);
    border-radius: 4px;
  }
</style>
