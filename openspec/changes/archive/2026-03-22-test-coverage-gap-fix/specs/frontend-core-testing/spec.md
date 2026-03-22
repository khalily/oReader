## ADDED Requirements

### Requirement: ErrorBoundary 组件测试
ErrorBoundary 组件 SHALL 正确捕获和处理 React 渲染错误。

#### Scenario: 捕获子组件错误
- **WHEN** 子组件抛出渲染错误
- **THEN** ErrorBoundary SHALL 捕获错误
- **AND** 显示 fallback UI 而不是崩溃

#### Scenario: onError 回调
- **WHEN** ErrorBoundary 捕获错误
- **THEN** onError 回调 SHALL 被调用
- **AND** 回调接收 Error 和 ErrorInfo 对象

#### Scenario: 重置错误状态
- **WHEN** 用户点击 "Try Again" 按钮
- **THEN** ErrorBoundary SHALL 清除错误状态
- **AND** 重新渲染子组件

#### Scenario: AppErrorBoundary 全屏错误
- **WHEN** AppErrorBoundary 捕获应用级错误
- **THEN** 显示全屏错误页面
- **AND** 提供 "Refresh Page" 和 "Dismiss" 按钮

### Requirement: ItemsPage 组件测试
ItemsPage 组件 SHALL 正确处理用户交互和状态管理。

#### Scenario: 渲染文章列表
- **WHEN** ItemsPage 加载并有文章数据
- **THEN** SHALL 显示文章列表
- **AND** 显示正确的未读计数

#### Scenario: 过滤器切换
- **WHEN** 用户切换过滤器（all/unread/starred/today）
- **THEN** URL 参数 SHALL 更新
- **AND** 文章列表 SHALL 根据过滤器重新加载

#### Scenario: 键盘导航 (j/k)
- **WHEN** 用户按下 'j' 键
- **THEN** 选中下一篇文章
- **WHEN** 用户按下 'k' 键
- **THEN** 选中上一篇文章

#### Scenario: 键盘操作 (s/r)
- **WHEN** 用户按下 's' 键
- **THEN** 切换当前文章的星标状态
- **WHEN** 用户按下 'r' 键
- **THEN** 切换当前文章的已读状态

#### Scenario: 无限滚动加载
- **WHEN** 用户滚动到列表底部
- **AND** hasMore 为 true
- **THEN** 触发加载更多文章
- **AND** 显示加载指示器

### Requirement: DeleteConfirmDialog 组件测试
DeleteConfirmDialog 组件 SHALL 正确处理删除确认流程。

#### Scenario: 显示确认对话框
- **WHEN** 对话框打开
- **THEN** 显示警告信息和确认/取消按钮

#### Scenario: 确认删除
- **WHEN** 用户点击确认按钮
- **THEN** onConfirm 回调 SHALL 被调用
- **AND** 对话框关闭

#### Scenario: 取消删除
- **WHEN** 用户点击取消按钮
- **THEN** onCancel 回调 SHALL 被调用
- **AND** 对话框关闭且不执行删除

### Requirement: ThemeContext 测试
ThemeContext SHALL 正确管理应用主题状态。

#### Scenario: 主题切换
- **WHEN** 用户切换主题（light/dark/system）
- **THEN** 主题状态 SHALL 更新
- **AND** localStorage SHALL 持久化主题设置

#### Scenario: 系统主题监听
- **WHEN** 主题设置为 system
- **AND** 操作系统主题变化
- **THEN** 应用主题 SHALL 自动跟随系统

### Requirement: useKeyboardShortcuts Hook 测试
useKeyboardShortcuts Hook SHALL 正确处理键盘快捷键。

#### Scenario: 忽略输入框中的快捷键
- **WHEN** 焦点在 input 或 textarea 元素上
- **THEN** 快捷键 SHALL NOT 触发

#### Scenario: enabled 状态控制
- **WHEN** enabled 设置为 false
- **THEN** 所有快捷键 SHALL 禁用

#### Scenario: preventDefault 行为
- **WHEN** 快捷键触发
- **THEN** 默认浏览器行为 SHALL 被阻止

### Requirement: itemsStore 测试
itemsStore SHALL 正确管理文章状态。

#### Scenario: 设置文章列表
- **WHEN** 调用 setItems
- **THEN** 文章列表 SHALL 更新

#### Scenario: 更新文章状态
- **WHEN** 调用 updateItemState 更新某文章的 is_read 或 is_starred
- **THEN** 对应文章的状态 SHALL 更新
- **AND** 其他文章状态保持不变

### Requirement: authStore 测试
authStore SHALL 正确管理认证状态。

#### Scenario: 设置认证状态
- **WHEN** 用户登录成功
- **THEN** isAuthenticated SHALL 为 true
- **AND** user 信息 SHALL 被存储

#### Scenario: 清除认证状态
- **WHEN** 用户登出
- **THEN** isAuthenticated SHALL 为 false
- **AND** user 信息 SHALL 被清除

#### Scenario: 状态持久化
- **WHEN** 页面刷新
- **THEN** 认证状态 SHALL 从 localStorage 恢复
