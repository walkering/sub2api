## 1. Dashboard 钻取入口

- [ ] 1.1 调整 `frontend/src/views/admin/DashboardView.vue`，让关键统计指标和副指标支持独立点击跳转，而不是只能静态展示。
- [ ] 1.2 为 dashboard 中的每个可点击指标建立明确的路由映射，包括正在使用的用户、今日新增用户、总用户数、API 秘钥、账号、今日请求等。
- [ ] 1.3 补充 dashboard 相关 i18n 与交互提示，确保管理员能识别哪些指标可钻取。

## 2. 目标页过滤契约

- [ ] 2.1 为 `frontend/src/views/admin/UsersView.vue` 增加 route query 初始化逻辑，并支持来自 dashboard 的 `activity_scope`、`created_scope` 等筛选参数。
- [ ] 2.2 扩展 `/api/v1/admin/users` 及其 service/repository 过滤模型，支持“今日活跃用户/正在使用的用户”后端筛选。
- [ ] 2.3 为 `frontend/src/views/admin/AccountsView.vue` 和 `frontend/src/views/admin/UsageView.vue` 补齐 route query 恢复逻辑，使账号状态和今日请求筛选在首屏生效。

## 3. 管理员 API 秘钥列表

- [ ] 3.1 新增管理员 API 秘钥列表路由、页面和前端 API 封装，承接 dashboard API 秘钥指标跳转。
- [ ] 3.2 在后端新增管理员 API 秘钥列表/搜索接口，至少支持总量列表、活跃状态筛选和基础搜索能力。
- [ ] 3.3 将 dashboard API 秘钥指标接入新的 `/admin/api-keys` 入口，并校验活跃/全部筛选语义与卡片统计一致。

## 4. 验证

- [ ] 4.1 为 dashboard 点击钻取、Users/Accounts/Usage query 恢复、API 秘钥列表筛选补充前端测试。
- [ ] 4.2 为用户活跃筛选和管理员 API 秘钥列表接口补充后端测试，覆盖筛选语义和分页结果。
- [ ] 4.3 手动验证从 `/admin/dashboard` 点击各指标后的落点、过滤结果和返回导航行为。
