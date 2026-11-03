# 05 前端 UI 规范（三端通用）

> 来源：从 V1（PHP 时代）三端前端代码中提取（基线提交 `8e4fd0f`，典型页面：`order/index.vue`、`order/detail`、`paypal-account/detail`、`paypal-account/components/balance-overview.vue`、`dashboard/index.vue`、`merchant/components/*`），叠加 EleAdmin Plus（`参考/ele-admin-plus-ruoyi-ultra`）的组件约定整理而成。
> 适用：`frontend/admin`、`frontend/merchant`、`frontend/agent` 的全部 V2 页面。与本规范冲突的旧写法以本规范为准。
> 全局样式落点：各端 `src/styles/index.scss`（`.app-section-card`、`.app-card-title`、`.page-section`、搜索卡片间距变量已存在，三端一致）。

---

## 0. 总原则

1. **只用两层容器**：`<ele-page>` 是页面唯一根容器；页面内一切内容都放在 `<ele-card>` 里，不允许裸 `<div>` 直接贴在 `ele-page` 下承载业务内容。
2. **颜色只用 Element 变量**：`var(--el-color-primary|success|warning|danger|info)`、`--el-text-color-*`、`--el-border-color-*`、`--el-fill-color-*`、`--el-bg-color*`。禁止写死 `#xxxxxx`（暗色主题与色弱模式依赖此约定）。
3. **间距只用三档**：页面分段 16px（`--app-page-section-gap`）、卡片内分组 12px、行内元素 6–8px。
4. **语义色映射固定**：成功/已完成 → `success`；处理中/待办/风险提示 → `warning`；失败/拒绝/关闭/高风险 → `danger`；中性/未开始/已归档 → `info`；主流程/选中 → `primary`。
5. **金额**：等宽数字（`font-variant-numeric: tabular-nums`）、加粗、始终带币种（前缀 `USD 12.34` 或后缀 `12.34 USD`，一个页面内只用一种）；空值显示 `-`，不显示 `0.00`（除非语义上确实为零）。
6. **脚本约定**：`<script setup>` + `defineOptions({ name })`；提示用 `EleMessage.*({ plain: true })`；确认用 `ElMessageBox.confirm(..., { type: 'warning', draggable: true })`；样式一律 `<style scoped>`（需要穿透用 `:deep()`）。

---

## 1. 页面骨架（五种页面类型）

| 类型 | 结构 | V1 参考 |
| --- | --- | --- |
| **列表页** | `ele-page` → [概览卡片行] → 搜索卡片 → 表格卡片 | `order/index.vue` |
| **详情页** | `ele-page hide-footer` → Hero 卡片（标题 + 状态标签 + 操作 + 摘要条） → [Tab 卡片] → 分区卡片（左 16 / 右 8 栅格） | `paypal-account/detail` |
| **表单页**（独立路由） | `ele-page hide-footer` → 表单 Hero → 分组表单卡片 → 底部操作栏卡片 | `paypal-account/form` + `account-edit.vue` |
| **工作台 / 看板** | `ele-page` → 筛选卡片 → 概览卡片行 → 图表卡片（16/8） → 排行/列表卡片 | `dashboard/index.vue` |
| **弹窗 / 抽屉** | `ele-modal`（表单 ≤ 680、详情 980）/ `el-drawer`（宽详情、带 Tab 时用抽屉） | `merchant-app-edit.vue`、`merchant-detail.vue` |

页面内相邻的顶层块之间用 `class="page-section"`（16px）隔开；**最后一个块（通常是表格卡片）加 `table-section`，取消底部外边距**。

---

## 2. 卡片规范（重点）

### 2.1 卡片类型总览

| 卡片 | 标识 | 用途 | 头部 | 内边距 |
| --- | --- | --- | --- | --- |
| **搜索卡片** | `<ele-card search-form>` | 列表筛选 | 无 | 全局样式接管：`18px 18px 10px`，控件高 34px、圆角 8px，`el-form-item` 下边距 14px |
| **表格卡片** | `<ele-card class="table-section" :body-style="{ paddingTop: '8px' }">` | 承载 `ele-pro-table` | 无（工具栏放 `#toolbar`） | 顶部 8px，其余默认 |
| **分区卡片** | `<ele-card class="app-section-card">` + `#header` 内 `.app-card-title` | 详情分区、看板图表、配置分组 | 标题 + 副标题 | 头部 `14px 18px 12px`、无下边线 |
| **概览卡片** | `<ele-card class="summary-card is-{type}">` 置于 `el-row/el-col` | 列表页顶部 / 看板 KPI | 无 | `16px 18px`（看板 `18px 20px`） |
| **Hero 卡片** | `<ele-card class="detail-hero">` | 详情页/表单页头部 | 自定义（标题行 + 描述 + 右侧操作） | `18px 20px` |
| **Tab 卡片** | `<ele-card :body-style="{ padding: '0 20px' }">` 内 `el-tabs` | 详情页多视图切换 | 无 | 上下 0 |

### 2.2 分区卡片标题（唯一允许的卡片标题写法）

```vue
<ele-card class="app-section-card">
  <template #header>
    <div class="app-card-title">
      <span>账号资料</span>
      <em>主体、归属与收单属性</em>
    </div>
  </template>
  ...
</ele-card>
```

- `span` 为标题（14px / 600），`em` 为副标题（12px / placeholder 色），`::before` 自动渲染 3×16 的主色竖条。
- 需要右侧操作（刷新、查看全部）时，把 `.app-card-title` 包进 `display:flex; justify-content:space-between` 的容器，按钮用 `<el-button link type="primary" :icon="Refresh">`。
- **禁止** `<ele-card header="xxx">` 字符串头部（无竖条、无副标题、与其他卡片风格不一致）。当前三端仍有 30 处（admin 19 / merchant 7 / agent 4）需改造，见 §6。

### 2.3 概览卡片（summary / stat）

两种形态，同一套视觉：

**A. 独立卡片行**（列表页顶部，可点击作为快捷筛选）

```vue
<el-row :gutter="16" class="page-section">
  <el-col v-for="item in cards" :key="item.key" :xs="24" :sm="12" :md="6" :lg="4">
    <ele-card :body-style="{ padding: '16px 18px' }" class="summary-card"
      :class="[`is-${item.type}`, { 'is-active': active === item.key }]" @click="item.onClick?.()">
      <div class="summary-head"><span class="summary-dot" /><span class="summary-label">{{ item.label }}</span></div>
      <div class="summary-value">{{ item.value }}</div>
      <div class="summary-desc">{{ item.desc }}</div>
    </ele-card>
  </el-col>
</el-row>
```

**B. 卡片内网格**（放在一个分区卡片里，如“余额汇总”）：`.stat-grid { display:grid; grid-template-columns: repeat(6, minmax(0,1fr)); gap:12px }`，子项 `.stat-card`，结构同上（`stat-head/stat-dot/stat-label/stat-value/stat-desc`）。

视觉规则（V1 原样）：
- 三行：`label`（13px secondary）→ `value`（22–24px / 700 / tabular-nums / 不换行）→ `desc`（12px secondary，单行省略）。
- 左上 8px 圆点 + 右下角 80–92px 的 6% 透明度同色圆形装饰（`::after`）。
- 颜色通过 `--summary-color` / `--stat-color` 变量按 `is-primary|success|warning|danger|info` 切换。
- **警示类（warning/danger）**：浅色底 `--el-color-*-light-9` + 描边 `--el-color-*-light-7`，label/value 用对应色，装饰透明度提高到 12%；有异常数据时加 `is-alert`，描边变实色。
- 可点击卡片：`cursor:pointer`，hover / `is-active` 时描边主色 + `--el-box-shadow-light`。
- 响应式：6 列 → ≤1400px 3 列 → ≤767px 2 列（网格形态）；卡片行形态用 `xs=24 sm=12 md=6 lg=4`。

### 2.4 Hero 卡片 + 摘要条（详情页）

```
┌ detail-hero ───────────────────────────────────────────────┐
│ [标题 18px/700]  [状态tag light] [子状态tag plain] [渠道badge]   [操作按钮组] │
│ 描述文字 13px secondary                                                │
│ ┌ summary-strip (grid, 1.2fr + 5×1fr, 圆角10, 渐变底, 单边框) ──────┐ │
│ │ 订单金额 | 美元记账金额 | 可退金额 | 商户 | 创建时间 | 支付时间 │ │
│ └───────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

- 摘要条项 `.summary-item`：上 `span` 12px secondary，下 `strong` 13px/600 单行省略；首项（金额）加 `is-amount`：主色 18px/700。项之间用左边线分隔，≤992px 变 2 列并改用上边线。
- 额度类摘要项可在 `strong` 下追加 `<el-progress :stroke-width="6" :show-text="false">`。
- Hero 内 `v-loading`；查无数据时在 Hero 内放 `<el-empty :image-size="90">`，不再渲染下方分区。
- 右上操作：主操作 `type="primary" plain`，返回 `plain`；放在 `el-space` 中。

### 2.5 分区卡片内容形态

| 内容 | 写法 |
| --- | --- |
| 键值信息 | `<el-descriptions :column="mobile ? 1 : 2|3" border class="detail-table">`；label 列固定宽 110px、常规字重；空值 `-`；跨列用 `:span` |
| 右侧窄栏键值 | `.side-kv`：`flex space-between`，`padding:10px 0`，虚线分隔，最后一项无线 |
| 指标小卡 | `.metric-grid`（2 列）+ `.metric-card`：`--el-fill-color-extra-light` 底、圆角 10、`strong` 20px，异常加 `is-danger` |
| 时间线 | `<el-timeline class="detail-timeline">`，每项 `log-content`(13px) + `log-operator`(12px secondary)；为空用 `<el-empty :image-size="70">` |
| 技术字段 | ID / 密钥 / IP / 哈希 用 `.mono`（Consolas, Monaco, monospace），密钥一律脱敏显示 |
| 错误文本 | `.error-text`（danger 色） |
| 提示 | 卡片底部 `<el-alert type="info" show-icon :closable="false">` |

### 2.6 表单卡片（独立表单页）

- 页面顶部 Hero：左侧 40px 圆角图标块 + 标题（16px/700）+ 描述；右侧 `text` 关闭按钮。
- 每个字段分组一个卡片，标题复用 `.app-card-title` 语义（V1 用 `.card-title + .section-bar`，V2 统一改用 `.app-card-title`）。
- 字段排布 `el-row :gutter="16"` + `el-col :span="12"`（移动端 24）；`label-width` 120px；开关/单选在同一行右侧补说明文字（12px secondary）。
- 底部操作栏独立卡片，右对齐：`取消`（plain）+ `保存`（primary，`:loading`）。
- 弹窗表单（≤ 6 个字段）不用分组卡片，直接 `ele-modal` + `el-form label-width 96px`。

---

## 3. 搜索卡片与表格

### 3.1 搜索卡片
- 必须是 `<ele-card search-form>`，独立组件 `components/xxx-search.vue`，向父组件 `emit('search', where)`。
- 栅格：输入项 `lg=6 md=8 sm=12 xs=24`；日期范围 `lg=8 md=12`；按钮列 `lg=4 md=8 sm=12`，`label-width="0"`。
- 控件：输入 `clearable placeholder="请输入"`，下拉 `clearable placeholder="请选择" class="ele-fluid"`，日期 `type="daterange" value-format="YYYY-MM-DD" class="ele-fluid"`。
- 回车触发搜索（`@keyup.enter`，`@submit.prevent`）；重置 = 清空 + 立即搜索。
- 按钮：`<el-button type="primary">搜索</el-button><el-button>重置</el-button>` 或 `btn-items` 的 `search/reset` 预设，一个端内只用一种。
- 筛选项 > 8 个时，默认只显示前两行，其余放入“展开”折叠（EleAdmin `ele-card` 的 `collapsable` 不用于搜索卡片）。

### 3.2 表单文案与默认值（提取自 V1 `*-search.vue` / `*-edit.vue`，36×“请选择”、34×“全部”为最高频写法）

**搜索表单**（筛选条件，v-model 为 `where.*`）

| 控件 | 规则 | 示例 |
| --- | --- | --- |
| 文本输入 | `v-model.trim` + `clearable` + `placeholder="请输入"`；**单号 / 邮箱类字段可给格式或匹配方式提示** | `请输入`、`PO…`、`精确或前缀`、`订单号、邮箱等`、`编号、名称、Stripe Account ID` |
| 下拉 | `clearable` + `class="ele-fluid"` + `placeholder="全部"`（可带维度：`全部渠道`、`全部环境`、`全部状态`）；默认值 `''` 表示不过滤 | 空值不发送该参数 |
| 日期范围 | `type="daterange"` `value-format="YYYY-MM-DD"` `range-separator="-"` `start-placeholder="开始日期"` `end-placeholder="结束日期"` `class="ele-fluid"`；默认空；按 `beginTime/endTime` 拆开提交 | 带时分秒用 `datetimerange` + `开始时间 / 结束时间` |
| 标签宽度 | 72–86px（中文 4–6 字），同一表单统一 | |
| 行为 | 回车即搜索；“重置”= 清空 + 立即搜索；无“查询条件必填” | |

**编辑表单**（弹窗 / 独立页，v-model 为 `form.*`）

| 控件 | 规则 | 示例 |
| --- | --- | --- |
| 文本输入 | `v-model.trim` + `clearable` + `placeholder="请输入{字段名}"`；有长度上限的加 `maxlength` + `show-word-limit` | `请输入企业名称` |
| 有格式约束的输入 | **用 placeholder 直接说明约束或给样例**，而不是写在 label 里 | `仅支持 gmail.com 或 outlook.com 邮箱`、`https://example.com/pay/callback`、`如 50000.00`、`输入 IP 后回车添加` |
| 下拉 | `placeholder="请选择{字段名}"`；非必填且有“不选”语义时写明 | `请选择经营国家`、`不选则暂不归属任何渠道` |
| 金额 | 文本输入（不用 `el-input-number`），`placeholder="如 100.00"`，右侧 `append` 币种；提交前 `toMinor()` | |
| 多行文本 | `type="textarea" :rows="2~3" maxlength show-word-limit`，placeholder 说明用途 | `可填写线下转账说明、收款备注等` |
| 开关 / 单选 | 右侧补 12px secondary 说明文字，说明开启后的效果 | |
| 默认值 | 新增时给业务合理默认（如 T+7、周结、保证金 180 天、状态启用）；编辑时回填；禁止把 `0`/空串当作“未设置”展示 | |
| 标签宽度 | 弹窗 96–110px；独立页 120px | |
| 密码 | `type="password" show-password`，placeholder 写策略：`10-64 位，须包含字母和数字` | |

**通用**
- 只读展示用 `el-descriptions`，不要用 `disabled` 的输入框。
- 必填项用 `rules` + `prop` 触发星号，不在 placeholder 里写“（必填）”。
- `el-input-number` 仅用于小范围整数（天数、bps、次数），并设置 `:min/:max`。
- 一次性批量校验脚本：`frontend/scripts/fix-form-hints.mjs`（按上下文补 placeholder / clearable / 日期提示；`--dry` 预览），新页面提交前运行一次。

### 3.3 表格
- `ele-pro-table`：`row-key`、`:columns`、`:datasource`、`:show-overflow-tooltip="true"`、`:highlight-current-row="true"`、`cache-key`（列设置记忆）。
- 列定义写在 `columns`（`computed` 或 `ref`），带 `slot` 的列**必须**同时给 `formatter`（导出/打印用）。
- 固定列：选择/序号 `fixed: 'left'`（宽 48–50，居中），操作列 `fixed: 'right'`、`hideInPrint/hideInExport`。
- 单号列：`<el-link type="primary" underline="never" @click="detail(row)">`；状态列：`<el-tag :type effect="light">`，子状态/辅助状态 `effect="plain"`；复合单元格（标签 + 文本）用 `inline-flex; gap:6px`，文本单行省略。
- 操作列：`el-space :size="6" spacer="|"` 内 `el-button link`（`primary/success/warning/danger` 对应语义），或 `btn-items type="link" divider`；更多操作收入 `more` 下拉。写操作必须带 `v-permission` / `permission`。
- 列表分页由 `ele-pro-table` 托管；**不要**在页面里再放 `el-pagination`（弹窗内嵌小列表除外）。

---

## 4. 状态标签与常量

- 每个业务域一个 `constants.js`：`xxxOptions = [{ value, label, type }]`，并导出 `getOptionLabel(options, v)` / `getOptionType(options, v)`；页面不内联状态文案。
- 环境标签：`live` → `danger plain`“Live 生产”，`sandbox` → `success plain`“Sandbox 沙箱”。
- 渠道标识：`stripe` → `primary`，`paypal` → `success`（V1 `PaymentProviderBadge` 约定）。
- 风险等级：low `success` / medium `warning` / high `danger`。

---

## 5. 响应式与可访问性

- 断点：`xs <768`、`sm ≥768`、`md ≥992`、`lg ≥1200`、看板网格额外 `1400`。
- 详情页在 `md` 以下：左右栏各 24；`el-descriptions` `:column` 由 `useMobile()` 决定 1 / 2。
- 所有可点击非按钮元素（概览卡片、单号链接）必须有 hover 反馈；颜色不是唯一信息载体（标签必须有文字）。
- 页面级 `v-loading` 只加在 Hero 或表格卡片，不在 `ele-page` 上整页遮罩。

---

## 6. 共享组件（三端同一份，`src/components/`，按需自动注册）

| 组件 | 用途 | 关键 API |
| --- | --- | --- |
| `<section-card>` | §2.2 分区卡片 | `title` / `subtitle`，插槽 `extra`（右侧操作）、`footer`；透传 `body-style` 与其他属性 |
| `<summary-cards>` | §2.3 概览卡片 | `items`（`key,label,value,desc,tag,type,alert,clickable,onClick`）、`span`（栅格）、`cols`（固定列数网格）、`inner`（卡片内网格）、`active`/`active-key`；事件 `select(item)` |
| `<detail-hero>` | §2.4 Hero + 摘要条 | `title` / `desc` / `summary`（`label,value,amount,progress`）、`card`（false 时用于抽屉/弹窗）；插槽 `tags`、`actions`、默认 |

全局工具类（`styles/index.scss`）：`.app-section-card` `.app-card-title` `.page-section` `.table-section` `.amount-text(.is-negative)` `.mono` `.error-text` `.muted-text`。

## 7. 整改记录（2026-10-18）

| 项 | 结果 |
| --- | --- |
| U1 字符串头部 | 30 处全部改为 `<section-card title subtitle>`，业务页补充了副标题（口径说明） |
| U2 概览卡片 | 三端 `SummaryCards` 共享组件；admin / merchant / agent 看板、销售渠道模块 5 个列表页接入；删除 `agent/components/summary-cards.vue`、`agent/shared/summary-cards.vue` 两套本地实现及其写死的 RGB 颜色 |
| U3 详情 Hero | `DetailHero` 共享组件；商户详情抽屉、总后台 / 商户端订单详情抽屉接入（标题 + 状态标签 + 操作 + 摘要条 + `app-card-title` 分区标题） |
| U4 页面分段 | 看板 `style="margin-top:16px"` 改为类；表格卡片补 `table-section` |
| U5 金额 | 全局 `.amount-text`；订单 / 退款 / 争议列表与详情的金额单元格统一包裹 |
| U6 技术字段 | 全局 `.mono`；上游单号列启用 |
| U8 表单文案 | 新增 §3.3；脚本整改 39 个文件：69+15+3 个无提示输入框补 `请输入` / `请输入{字段}` + `clearable`，6 个下拉补 `全部` / `请选择{字段}`，8 个日期范围补起止提示与分隔符 |
| U7 搜索卡片 | 待办：`developer/webhook` 推送记录等仍把筛选放在卡片内 inline 表单，下一轮拆分 |
| U9 商户列表（2026-10-23） | 自定义统计条（裸 `div`、位于搜索卡片下方）改为 `<summary-cards>` 置顶，6 张卡片（全部 / 审核中 / 观察中 / 已暂停 / 高风险 / 经营范围待补审）点击即快捷筛选，筛选值写回搜索卡片（新增"经营范围"筛选项，条件始终可见）；搜索卡片拆为 `merchant/components/merchant-search.vue`（`btn-items` search / reset 预设）；表格卡片补 `table-section`、`highlight-current-row`，带 slot 的列补 `formatter`，操作列 `hideInPrint/hideInExport`，商户号列左固定 |
| SummaryCards 换行间距 | 三端组件 `.summary-row` 补 `row-gap: 16px`：卡片行在窄屏换行时上下不再贴边 |
| U10 总后台列表（2026-10-24） | 列表页统一：表格卡片 `table-section` + `highlight-current-row`，带 slot 的列补 `formatter`，操作列 `fixed right` + `hideInPrint/hideInExport`，操作格改为 `el-space` 分隔的 `el-button link`（已是 `btn-items type="link" divider` 的保持）；搜索按钮统一为 `btn-items` 的搜索/重置，日期范围用开始日期/结束日期；确认框补 `draggable`；状态与环境标签改走常量（环境为 Live 生产 / Sandbox 沙箱） |
