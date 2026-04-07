## 1. 设置模型与后台配置

- [x] 1.1 为后端设置常量、`SystemSettings`、DTO 与前端 `SystemSettings/UpdateSettingsRequest` 新增 `linuxdo_connect_gift_subscriptions` 字段，并沿用默认订阅的数组结构。
- [x] 1.2 在设置读写、初始化、归一化与合法性校验中复用默认订阅规则，确保空列表合法、默认值为 `[]`，并补充相关单元测试。
- [x] 1.3 在管理后台设置页和中英文文案中新增 LinuxDo Connect 赠送订阅编辑区域，支持空状态展示与保存。

## 2. LinuxDo OAuth 发放逻辑

- [x] 2.1 在 `AuthService` 中新增 LinuxDo Connect 专属赠送订阅发放辅助方法，只在首次创建 LinuxDo OAuth 用户后调用。
- [x] 2.2 覆盖 LinuxDo OAuth 直接注册和邀请码补全注册两条成功建号路径，并继续使用 `AssignOrExtendSubscription` 处理分配或续期。
- [x] 2.3 增加测试，验证首次注册发放、空配置不发放、已有账号再次登录不重复发放，以及邀请码补全注册同样发放。
