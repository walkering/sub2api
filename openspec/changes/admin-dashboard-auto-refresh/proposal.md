## 为什么

`/admin/dashboard` 当前只支持手动刷新。管理员在观察请求量、费用、用户活跃度和排行榜时，需要频繁重复点击刷新，既影响使用效率，也容易错过短时间内的数据变化。现在补充自动刷新能力，可以让总览页在保持当前筛选条件的前提下持续更新。

## 变更内容

- 在 `/admin/dashboard` 增加自动刷新开关，并允许管理员选择固定刷新间隔。
- 自动刷新时复用现有仪表盘加载逻辑，刷新统计卡片、趋势图和排行榜数据，同时显示倒计时或当前状态。
- 自动刷新配置在前端本地持久化，刷新页面后沿用上次选择；默认保持关闭，避免无意增加请求量。
- 手动刷新、时间范围变更和粒度切换后，自动刷新倒计时应重新开始，避免连续重复请求。

## 功能 (Capabilities)

### 新增功能
- `admin-dashboard-auto-refresh`: 管理后台总览页支持自动刷新、刷新间隔选择和倒计时展示，并对当前筛选条件持续生效。

### 修改功能
- 

## 影响

- 前端页面：`frontend/src/views/admin/DashboardView.vue`
- 前端文案与交互：`frontend/src/views/admin` 下相关组件与 i18n 文案
- 测试：`frontend/src/views/admin/__tests__/DashboardView.spec.ts`
- API：继续使用现有 `adminAPI.dashboard.getSnapshotV2`、`getUserUsageTrend`、`getUserSpendingRanking`，不新增后端接口
