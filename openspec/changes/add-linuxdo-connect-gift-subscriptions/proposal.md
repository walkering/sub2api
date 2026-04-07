## 为什么

当前系统只支持为所有新用户统一配置默认订阅，无法单独控制 LinuxDo Connect 首次登录用户的赠送订阅。需要增加一个可为空、默认不赠送的 LinuxDo Connect 专属订阅配置，便于按渠道运营而不影响其他注册入口。

## 变更内容

- 在管理后台系统设置中新增 `LinuxDo Connect 赠送订阅` 配置，结构沿用现有默认订阅列表（`group_id` + `validity_days`）。
- 允许该配置为空；空值和默认值都表示不向 LinuxDo Connect 登录用户赠送任何订阅。
- 仅在 LinuxDo Connect 首次成功创建本地用户时发放这些赠送订阅；已有账号再次通过 LinuxDo Connect 登录时不重复发放。
- 赠送订阅的分组合法性、去重和有效期归一化规则与现有默认订阅保持一致。

## 功能 (Capabilities)

### 新增功能
- `linuxdo-connect-gift-subscriptions`: 管理 LinuxDo Connect 登录专属赠送订阅的配置与首次注册发放行为。

### 修改功能

## 影响

- 后端系统设置模型、设置读写与校验逻辑
- LinuxDo Connect OAuth 首次注册流程
- 管理后台设置 API、设置页表单与文案
- 相关单元测试与集成测试
