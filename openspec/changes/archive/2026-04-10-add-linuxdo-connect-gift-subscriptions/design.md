## 上下文

系统已经支持 `default_subscriptions`，会在普通注册、管理员创建用户和 LinuxDo Connect 首次 OAuth 注册时统一发放默认订阅。当前 LinuxDo Connect 设置仅包含开关、Client ID、Client Secret 和回调地址，缺少渠道专属赠送订阅配置。此次需求要求新增一组独立设置，并且默认空列表、不发放任何赠送订阅。

LinuxDo Connect 注册路径存在两个成功创建用户的分支：一个是直接完成 OAuth 登录，一个是启用邀请码时通过 `complete-registration` 补全注册。设计必须覆盖这两个分支，同时避免对已有账号的重复发放。

## 目标 / 非目标

**目标：**
- 新增 `linuxdo_connect_gift_subscriptions` 设置，数据结构与 `default_subscriptions` 保持一致。
- 允许管理员将该设置留空，且系统默认值为 `[]`。
- 仅在 LinuxDo Connect 首次成功创建本地用户后发放专属赠送订阅。
- 复用现有订阅分组校验、有效期归一化和分配能力，减少重复实现。

**非目标：**
- 不改变现有 `default_subscriptions` 的行为。
- 不向普通邮箱注册、管理员创建用户或已有账号再次登录补发 LinuxDo Connect 赠送订阅。
- 不新增公开设置接口；该配置仅供管理员管理。

## 决策

### 决策 1：新增独立设置键并复用默认订阅的数据结构

新增 `SettingKeyLinuxDoConnectGiftSubscriptions`，在 `SystemSettings`、DTO、前端 API 类型与管理后台表单中引入 `linuxdo_connect_gift_subscriptions` 字段，字段值使用与 `default_subscriptions` 相同的 JSON 数组结构。

原因：
- 现有前后端已经有订阅列表编辑与序列化模式，复用成本最低。
- 独立字段能清晰表达“LinuxDo 渠道专属”语义，避免和全局默认订阅混淆。

备选方案：
- 复用 `default_subscriptions` 并增加条件开关：语义不清晰，也会让普通注册路径受到影响。
- 只支持单个赠送订阅：灵活性不足，与现有默认订阅模型不一致。

### 决策 2：新增专用发放辅助方法，只在新建 LinuxDo 用户后调用

在 `AuthService` 中新增与 `assignDefaultSubscriptions` 平行的 LinuxDo 专属发放辅助方法，并仅在 LinuxDo OAuth 首次创建用户成功后调用。直接登录创建用户和邀请码补全注册两条路径都必须调用该方法；已有账号登录路径不调用。

原因：
- 能明确区分“默认订阅”和“LinuxDo 赠送订阅”的触发条件。
- 便于在测试中验证首次注册与重复登录的差异行为。

备选方案：
- 在 `assignDefaultSubscriptions` 中加入来源参数：会把两类逻辑耦合在一起，降低可读性。
- 每次 LinuxDo 登录都尝试发放并依赖幂等：虽然可行，但会带来不必要的续期风险。

### 决策 3：发放时继续复用 `AssignOrExtendSubscription`

LinuxDo 赠送订阅的每一项发放都继续调用现有 `AssignOrExtendSubscription`。当 LinuxDo 赠送订阅与全局默认订阅配置包含同一分组时，系统沿用当前语义，对该订阅执行创建或续期，而不是创建重复记录。

原因：
- 现有分配接口已经具备幂等和续期语义，适合处理同一用户同一分组的发放。
- 避免新增一套特殊冲突处理逻辑。

备选方案：
- 改用严格不允许重复的 `AssignSubscription`：会在默认订阅与 LinuxDo 赠送订阅重叠时产生冲突。
- 检测重叠后跳过 LinuxDo 发放：会让管理员配置结果难以理解。

## 风险 / 权衡

- [默认订阅与 LinuxDo 赠送订阅重叠时会叠加有效期] -> 通过复用 `AssignOrExtendSubscription` 保持行为一致，并在文案或注释中明确该语义。
- [LinuxDo OAuth 新用户创建存在多条分支，容易漏掉其一] -> 把发放逻辑收敛到独立辅助方法，并为直接注册和邀请码补全都补测试。
- [设置校验逻辑分叉导致前后端行为不一致] -> 复用现有默认订阅的归一化和分组合法性校验模式。

## Migration Plan

1. 为设置存储新增 `linuxdo_connect_gift_subscriptions` 键，初始化默认值为 `[]`。
2. 部署后管理员可按需配置该字段；未配置时系统保持现状，不会给 LinuxDo 登录用户新增订阅。
3. 若需回滚，保留数据库中的该设置键不会影响旧版本继续运行；旧版本会忽略未知设置。

## Open Questions

- 暂无阻塞性开放问题；实现时只需确认后台提示文案是否需要明确说明“与默认订阅重叠时会顺延有效期”。
