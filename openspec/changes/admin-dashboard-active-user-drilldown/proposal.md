## 为什么

`/admin/dashboard` 里的统计卡片目前只能看，不能顺着指标继续钻取。管理员看到“正在使用的用户”“API 秘钥”“账号”“今日请求”等数据后，还需要手动切到其他页面重新筛选，路径长且容易筛错。

这次变更要把总览页变成可操作入口，让关键指标支持一键进入对应管理列表，并自动带上与指标语义一致的过滤条件。

## 变更内容

- 为 `/admin/dashboard` 的关键统计指标和副指标增加点击跳转能力，包括正在使用的用户、API 秘钥、账号、今日请求、今日新增用户、总用户数等。
- 明确每个指标的跳转目标页和 query 过滤契约，让目标页在首次加载时就能恢复来自仪表盘的筛选条件。
- 为“正在使用的用户”补充后台可执行的筛选能力，使 `/admin/users` 可以直接过滤出当日有请求的用户。
- 新增管理员 API 秘钥管理列表页和对应后台列表接口，作为仪表盘 API 秘钥指标的钻取目的地。

## 功能 (Capabilities)

### 新增功能
- `admin-dashboard-entity-drilldown`: 管理后台总览页支持从统计指标直接进入对应管理列表或用量页，并自动应用与指标一致的过滤条件。

### 修改功能

## 影响

- 前端页面：`frontend/src/views/admin/DashboardView.vue`
- 前端路由与目标页：`frontend/src/router/index.ts`、`frontend/src/views/admin/UsersView.vue`、`frontend/src/views/admin/AccountsView.vue`、`frontend/src/views/admin/UsageView.vue`
- 前端新页面与 API：管理员 API 秘钥列表页、`frontend/src/api/admin` 下相关接口、i18n 文案与测试
- 后端接口：`/api/v1/admin/users` 过滤条件扩展、管理员 API 秘钥列表/搜索接口、相关 service/repository 查询逻辑
- 数据层：默认不要求迁移；若实现阶段验证需要新增索引以支撑活跃用户/API 秘钥筛选性能，再单独补充
