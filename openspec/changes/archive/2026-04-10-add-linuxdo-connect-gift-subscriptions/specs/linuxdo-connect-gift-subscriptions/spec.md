## 新增需求

### 需求:管理员可以配置 LinuxDo Connect 赠送订阅
系统必须允许管理员为 LinuxDo Connect 登录配置专属赠送订阅列表。每个列表项必须包含 `group_id` 和 `validity_days`，并且只能引用订阅类型分组；重复分组或无效配置必须被拒绝。该配置可以为空，空值或未配置时系统必须视为“不赠送订阅”。

#### 场景:保存空赠送订阅列表
- **当** 管理员在系统设置中未添加任何 LinuxDo Connect 赠送订阅并保存
- **那么** 系统必须保存空列表
- **并且** 后续 LinuxDo Connect 首次注册流程必须不自动分配任何专属赠送订阅

#### 场景:拒绝非法赠送订阅配置
- **当** 管理员提交重复分组、非订阅类型分组或无效 `group_id` / `validity_days` 的 LinuxDo Connect 赠送订阅配置
- **那么** 系统必须拒绝保存该配置

#### 场景:读取已保存的赠送订阅配置
- **当** 管理员重新获取系统设置
- **那么** 系统必须返回已保存的 LinuxDo Connect 赠送订阅列表供再次编辑

### 需求:系统仅在首次 LinuxDo Connect 注册时发放赠送订阅
系统必须仅在 LinuxDo Connect 首次成功创建本地用户后发放专属赠送订阅。对于已存在本地账号的 LinuxDo Connect 登录，系统禁止重复发放该赠送订阅。

#### 场景:首次 LinuxDo Connect 注册发放赠送订阅
- **当** 用户通过 LinuxDo Connect 首次完成 OAuth 注册且系统已配置赠送订阅列表
- **那么** 系统必须在用户创建成功后为该用户分配每一项 LinuxDo Connect 赠送订阅

#### 场景:邀请码补全注册时发放赠送订阅
- **当** LinuxDo Connect 用户在邀请码模式下完成 `complete-registration` 并成功创建本地用户
- **那么** 系统必须按已配置的赠送订阅列表发放赠送订阅

#### 场景:首次 LinuxDo Connect 注册但未配置赠送订阅
- **当** 用户通过 LinuxDo Connect 首次完成 OAuth 注册且赠送订阅列表为空
- **那么** 系统必须完成注册和登录流程
- **并且** 系统必须不创建任何 LinuxDo Connect 专属赠送订阅记录

#### 场景:已有账号再次通过 LinuxDo Connect 登录
- **当** 已存在本地账号的用户再次通过 LinuxDo Connect 登录
- **那么** 系统必须直接完成登录
- **并且** 系统必须不重复发放 LinuxDo Connect 专属赠送订阅

## 修改需求

## 移除需求
