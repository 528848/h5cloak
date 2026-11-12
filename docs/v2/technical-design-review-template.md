# 默认模板模块技术方案

配套产品文档：`docs/v2/prd-review-template.md`。本文只描述怎么改，不在这一步改代码。

## 1. 做法

平台模板增加来源类型，快照把类型和内容带给网关。网关在「审核」这一支按类型处理：

| 类型 | 网关行为 | 复用 |
| --- | --- | --- |
| `html` | 渲染模板，任意路径同一份 HTML | 现有 `proxy.renderTemplate` |
| `url` | 反向代理到模板根地址，保留路径和查询 | 现有审核源站回源 |
| `static` | 在模板目录内按路径读文件 | 现有 `serveStatic`，缺文件默认 404 |

站点审核来源 `origin` / `template` / `static` 的枚举不改。`template` 不再默认等于「一段 HTML」，而是指向一条平台模板，由模板自己的 `kind` 决定出站方式。站点级审核源站和站点级静态包的代码路径不动。

## 2. 数据

`review_templates` 增加字段，已有行视为 `kind=html`：

| 字段 | 说明 |
| --- | --- |
| `kind` | `html` / `url` / `static`，默认 `html` |
| `status` | `enabled` / `disabled`，默认 `enabled` |
| `url` | 网址模板的根地址，可空 |
| `host_header` | 回源 Host，可空 |
| `timeout_ms` | 默认 8000 |
| `entry` | 静态入口，默认 `index.html` |
| `spa_fallback` | 静态项目缺文件时是否回入口，默认 0 |
| `active_version` | 静态项目当前版本，HTML/网址为 0 |
| `updated_at` | 保存时间 |

`html` 列保留。网址和静态模板该列为空字符串。

新表 `review_template_files`：

| 字段 | 说明 |
| --- | --- |
| `id` | 主键 |
| `template_id` | 模板 |
| `version` | 从 1 递增 |
| `dir` | 解压目录，相对数据根，例如 `templates/12/v3` |
| `entry` | 该版本入口 |
| `bytes` | 解压后总大小 |
| `file_count` | 文件数 |
| `status` | `ready` / `archived` |
| `created_at` | 上传时间 |

同一模板只保留最近 5 个 `ready`/`archived` 版本，更旧的版本删目录和行。当前使用的版本由 `review_templates.active_version` 指向。

不把静态文件放进 MySQL。目录放在现有数据根下：本地 `./.local/templates`，服务器 `/opt/cloak/data/templates`。

## 3. 上传与校验

上传接口只接受 `multipart` 的 `.zip`，上限 50MB。解压到临时目录，通过校验后再改名为版本目录。

校验：

- 压缩包内路径用 `path.Clean`，拒绝绝对路径、`..`、空路径和符号链接。
- 解压后文件数 ≤ 2000，单文件 ≤ 20MB，总大小 ≤ 80MB。
- 入口文件必须存在。未指定时依次找 `index.html`、`index.htm`。找不到则整包拒绝并删临时目录。
- 不允许把压缩包解到数据根之外。

网址校验在保存时做：只允许 `http`/`https`；解析主机后拒绝环回、链路本地、私网、云元数据地址。探测用 `HEAD`，失败再 `GET`，超时取模板上的 `timeout_ms`。探测失败不阻止保存，把结果写进操作日志，接口返回「已保存，探测失败」和原因。

HTML 保存前用 `html/template` 解析。失败返回 400 和解析错误，不写库。

## 4. 管理接口

仍挂在总后台 `/api/admin/template`，权限不变。

| 方法 | 路径 | 作用 |
| --- | --- | --- |
| GET | `/template/list` | 分页。查询参数：`keyword`、`kind`、`status`。每行带引用站点数 |
| GET | `/template/:id` | 详情。静态项目附当前版本和前两层文件名 |
| POST | `/template` | 新建 HTML 或网址 |
| PUT | `/template/:id` | 修改。改 `kind` 且引用数 > 0 时，请求体要带 `confirmKindChange=true`，否则 409 |
| DELETE | `/template/:id` | 引用数 > 0 返回 409。同时删版本目录 |
| POST | `/template/:id/disable` 与 `/enable` | 停用、启用 |
| POST | `/template/:id/package` | 上传新版本，成功后把 `active_version` 指到它 |
| POST | `/template/:id/rollback` | 体里带 `version`，只能滚到仍保留的版本 |
| POST | `/template/:id/preview` | HTML 用请求里的变量渲染后返回；网址返回探测状态码和耗时 |

列表从「一次返回全部」改为分页，前端表格才能按规范接搜索。

引用站点数来自 `site_review_contents.template_id` 且 `source=template`。

保存、上传、回滚、停用、删除成功后，对引用这些模板的站点逐个发布快照。发布方式沿用现在的 `cfg:changed`。

## 5. 快照

`snapshot.Review` 增加可选字段，旧快照没有这些字段时按 HTML 模板处理：

```text
templateKind      html | url | static
templateURL
templateHost
templateTimeoutMs
staticDir
staticEntry
staticSpaFallback
```

`template` 来源构建快照时：

- 模板不存在或已停用：`templateKind=html`，HTML 用现有内置占位页，变量仍用站点自己的。
- `html`：把模板 HTML 和站点变量放进快照，与现在相同。
- `url`：写入根地址、Host、超时。不在构建时抓整站。
- `static`：写入当前版本的绝对目录、入口、是否单页回退。

站点 `static` 来源继续写站点自己的 `static_packages` 目录，`staticSpaFallback=true`，保持现有缺文件回入口的行为。

## 6. 网关出站

判定结果是审核且来源是模板时：

- `html`：走现有渲染，加 `X-Robots-Tag: noindex`。
- `url`：用现有反向代理。上游 URL 为模板根地址拼上访客路径和原始查询。Host 用 `templateHost`，空则用根地址的主机。超时用 `templateTimeoutMs`。上游失败时走站点现有的失败兜底，不把上游错误页原样当成成功。
- `static`：根目录限制在 `staticDir`。目录请求改找入口文件。文件用 `http.ServeContent`。`staticSpaFallback=false` 且文件不存在时返回 404 页面，不再回入口。为 `true` 时保持现在的回入口逻辑。同样加 `noindex`。

开放接口的 `renderHint`：

- `html`：继续返回渲染后的 HTML。
- `url`：返回 `type=url` 和根地址，不抓页面。
- `static`：返回 `type=static`。托管入口由网关出文件；服务端接入不下载整包。

## 7. 总后台页面

按 `docs/前端UI规范.md` 的列表页和独立表单页，不沿用现在的 `el-dialog`。

- 列表：`summary-cards` 四张卡对应全部和三种来源，点击把 `kind` 写入搜索条件。搜索卡片含关键字、来源、状态。表格用 `ele-pro-table`，操作列固定在右侧，删除和停用用确认框。
- 表单路由：`/admin/template/form` 新建，`/admin/template/form/:id` 编辑。基本信息一张分区卡，来源内容第二张卡。底部独立操作栏：取消返回列表，保存提交。
- 来源切换：若 HTML、网址或已选文件有内容，先确认再切换。
- 静态版本：表单里用表格列出版本、大小、时间，当前版本打标签。回滚前确认。

商户站点的审核配置只改「平台模板」这一支：下拉显示名称和来源；HTML 才出现变量和预览；网址、静态项目只读说明。

## 8. 迁移与兼容

1. 迁移给 `review_templates` 加列，并把已有行写成 `kind=html`、`status=enabled`。
2. 不改 `site_review_contents` 的行。已选用旧模板的站点，下一次发布快照后仍走 HTML 分支。
3. 网关先部署能识别新快照字段的版本，再开放新类型的保存。旧网关忽略未知字段，只会把网址和静态模板错当成空 HTML，所以不能颠倒顺序。

## 9. 测试

- 模板服务：三种来源的保存校验、内网地址拒绝、HTML 语法错误、压缩包穿越和入口缺失、引用中的删除返回 409、版本只留 5 个。
- 快照：停用模板得到占位 HTML；静态模板目录指向当前版本。
- 代理：网址回源保留路径；静态文件命中、目录入口、缺文件 404、打开回退后缺文件返回入口；站点自己的静态包仍然回入口。
- 页面：列表筛选、表单三种来源切换确认、删除确认。不在这一步做压测。
