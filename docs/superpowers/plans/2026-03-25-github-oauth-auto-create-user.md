# GitHub OAuth 自动创建用户 - 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 改进 GitHub OAuth 流程，支持无邮箱用户自动获取邮箱、自动关联账户、保存 GitHub 登录名

**Architecture:**
- 新增 `PendingOAuth` 模型存储待确认的 OAuth 数据（5分钟过期）
- 扩展 `OAuthHandler` 添加 `/user/emails` API 调用和两个新端点
- 前端新增 `/oauth/pending` 页面收集用户邮箱

**Tech Stack:** Go 1.21+, GORM, React 18, React Router, react-hook-form, zod

---

## 文件结构

### 新增文件
| 文件 | 职责 |
|------|------|
| `internal/repository/pending_oauth_repository.go` | PendingOAuth 数据访问层实现 |
| `internal/repository/pending_oauth_repository_test.go` | Repository 单元测试 |
| `web/src/pages/oauth/OAuthPendingPage.tsx` | 邮箱输入页面组件 |

### 修改文件
| 文件 | 变更内容 |
|------|----------|
| `internal/model/models.go` | User 添加 `GitHubLogin` 字段；新增 `PendingOAuth` 模型 |
| `internal/service/interfaces.go` | 添加 `PendingOAuthRepository` 接口 |
| `internal/handler/oauth_handler.go` | 添加 `getBestEmail` 函数、`/user/emails` 调用、`pendingOAuthRepo` 依赖、两个新端点 |
| `internal/handler/oauth_handler_test.go` | 新增测试用例覆盖新流程 |
| `cmd/server/main.go` | 初始化 `PendingOAuthRepository`、更新依赖注入、添加新路由 |
| `web/src/App.tsx` | 添加 `/oauth/pending` 路由 |

---

## Task 1: 数据模型扩展

**Files:**
- Modify: `internal/model/models.go:20-35`

- [ ] **Step 1: 在 User 模型中添加 GitHubLogin 字段**

在 `User` 结构体中添加新字段：

```go
// User 模型 - 在 GitHubID 字段后添加
type User struct {
    Base
    Email        string `gorm:"uniqueIndex;type:varchar(255)" json:"email"`
    PasswordHash string `gorm:"type:varchar(255)" json:"-"`
    Nickname     string `gorm:"type:varchar(100)" json:"nickname"`
    AvatarURL    string `gorm:"type:varchar(500)" json:"avatar_url"`
    AuthProvider string `gorm:"type:varchar(50);default:'email'" json:"auth_provider"`
    GitHubID     string `gorm:"type:varchar(100)" json:"github_id,omitempty"`
    GitHubLogin  string `gorm:"type:varchar(100)" json:"github_login,omitempty"` // 新增
}
```

- [ ] **Step 2: 添加 PendingOAuth 模型**

在 `models.go` 文件末尾添加新模型：

```go
// PendingOAuth 存储待确认的 OAuth 数据（5分钟过期）
type PendingOAuth struct {
    Base
    Token       string    `gorm:"type:varchar(64);not null;uniqueIndex" json:"token"`
    GitHubID    string    `gorm:"type:varchar(100);not null" json:"github_id"`
    GitHubLogin string    `gorm:"type:varchar(100);not null" json:"github_login"`
    Nickname    string    `gorm:"type:varchar(100)" json:"nickname"`
    AvatarURL   string    `gorm:"type:varchar(500)" json:"avatar_url"`
    ExpiresAt   time.Time `gorm:"not null" json:"expires_at"`
}

// IsExpired 判断是否过期
func (p *PendingOAuth) IsExpired() bool {
    return time.Now().After(p.ExpiresAt)
}
```

- [ ] **Step 3: 运行测试验证模型变更**

Run: `go test ./internal/model/... -v`
Expected: PASS（如果没有现有测试，确认编译通过即可）

- [ ] **Step 4: 提交模型变更**

```bash
git add internal/model/models.go
git commit -m "feat(model): add GitHubLogin to User and PendingOAuth model"
```

---

## Task 2: 添加 PendingOAuthRepository 接口

**Files:**
- Modify: `internal/service/interfaces.go`

- [ ] **Step 1: 添加 PendingOAuthRepository 接口定义**

在 `interfaces.go` 中添加新接口（在 `OAuthStateRepository` 接口之后）：

```go
// PendingOAuthRepository 定义待确认 OAuth 数据访问接口
type PendingOAuthRepository interface {
    Create(ctx context.Context, pending *model.PendingOAuth) error
    GetByToken(ctx context.Context, token string) (*model.PendingOAuth, error)
    Delete(ctx context.Context, token string) error
}
```

- [ ] **Step 2: 验证编译通过**

Run: `go build ./internal/service/...`
Expected: 编译成功，无错误

- [ ] **Step 3: 提交接口变更**

```bash
git add internal/service/interfaces.go
git commit -m "feat(service): add PendingOAuthRepository interface"
```

---

## Task 3: 实现 PendingOAuthRepository

**Files:**
- Create: `internal/repository/pending_oauth_repository.go`
- Create: `internal/repository/pending_oauth_repository_test.go`

- [ ] **Step 1: 编写 Repository 实现失败的测试**

创建 `internal/repository/pending_oauth_repository_test.go`：

```go
package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"oReader/internal/model"
)

func setupPendingOAuthTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&model.PendingOAuth{})
	require.NoError(t, err)

	return db
}

func TestPendingOAuthRepository_Create(t *testing.T) {
	db := setupPendingOAuthTestDB(t)
	repo := NewPendingOAuthRepository(db)
	ctx := context.Background()

	pending := &model.PendingOAuth{
		Token:       "test-token-64-chars-padding-padding-padding-padding-padding-padding",
		GitHubID:    "12345",
		GitHubLogin: "testuser",
		Nickname:    "Test User",
		AvatarURL:   "https://example.com/avatar.png",
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	}

	err := repo.Create(ctx, pending)
	assert.NoError(t, err)
	assert.NotEmpty(t, pending.ID)
}

func TestPendingOAuthRepository_GetByToken(t *testing.T) {
	db := setupPendingOAuthTestDB(t)
	repo := NewPendingOAuthRepository(db)
	ctx := context.Background()

	// 创建测试数据
	pending := &model.PendingOAuth{
		Token:       "test-token-64-chars-padding-padding-padding-padding-padding-padding",
		GitHubID:    "12345",
		GitHubLogin: "testuser",
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	}
	require.NoError(t, repo.Create(ctx, pending))

	// 测试查找
	found, err := repo.GetByToken(ctx, pending.Token)
	assert.NoError(t, err)
	assert.Equal(t, pending.GitHubID, found.GitHubID)
	assert.Equal(t, pending.GitHubLogin, found.GitHubLogin)

	// 测试找不到
	_, err = repo.GetByToken(ctx, "non-existent-token-padding-padding-padding-padding-padding")
	assert.Error(t, err)
}

func TestPendingOAuthRepository_Delete(t *testing.T) {
	db := setupPendingOAuthTestDB(t)
	repo := NewPendingOAuthRepository(db)
	ctx := context.Background()

	// 创建测试数据
	pending := &model.PendingOAuth{
		Token:       "test-token-64-chars-padding-padding-padding-padding-padding-padding",
		GitHubID:    "12345",
		GitHubLogin: "testuser",
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	}
	require.NoError(t, repo.Create(ctx, pending))

	// 删除
	err := repo.Delete(ctx, pending.Token)
	assert.NoError(t, err)

	// 验证已删除
	_, err = repo.GetByToken(ctx, pending.Token)
	assert.Error(t, err)
}

func TestPendingOAuth_IsExpired(t *testing.T) {
	// 测试未过期
	pending := &model.PendingOAuth{
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	assert.False(t, pending.IsExpired())

	// 测试已过期
	pending.ExpiresAt = time.Now().Add(-1 * time.Minute)
	assert.True(t, pending.IsExpired())
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/repository/pending_oauth_repository_test.go -v`
Expected: FAIL - 函数未定义

- [ ] **Step 3: 实现 PendingOAuthRepository**

创建 `internal/repository/pending_oauth_repository.go`：

```go
package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"oReader/internal/model"
	"oReader/internal/service"
)

type pendingOAuthRepository struct {
	db *gorm.DB
}

// NewPendingOAuthRepository 创建 PendingOAuthRepository 实例
func NewPendingOAuthRepository(db *gorm.DB) service.PendingOAuthRepository {
	return &pendingOAuthRepository{db: db}
}

func (r *pendingOAuthRepository) Create(ctx context.Context, pending *model.PendingOAuth) error {
	return r.db.WithContext(ctx).Create(pending).Error
}

func (r *pendingOAuthRepository) GetByToken(ctx context.Context, token string) (*model.PendingOAuth, error) {
	var pending model.PendingOAuth
	err := r.db.WithContext(ctx).Where("token = ?", token).First(&pending).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("pending oauth not found")
		}
		return nil, err
	}
	return &pending, nil
}

func (r *pendingOAuthRepository) Delete(ctx context.Context, token string) error {
	return r.db.WithContext(ctx).Where("token = ?", token).Delete(&model.PendingOAuth{}).Error
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `go test ./internal/repository/pending_oauth_repository_test.go -v`
Expected: PASS - 所有测试通过

- [ ] **Step 5: 提交 Repository 实现**

```bash
git add internal/repository/pending_oauth_repository.go internal/repository/pending_oauth_repository_test.go
git commit -m "feat(repository): implement PendingOAuthRepository with tests"
```

---

## Task 4: 扩展 OAuthHandler - 添加辅助函数

**Files:**
- Modify: `internal/handler/oauth_handler.go`

- [ ] **Step 1: 添加 GitHubEmail 结构体和 getBestEmail 函数**

在 `oauth_handler.go` 文件中，在 `GitHubUser` 结构体之后添加：

```go
// GitHubEmail 表示 /user/emails 返回的邮箱项
type GitHubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// getBestEmail 选择最佳邮箱
// 优先级: primary && verified > verified > primary > 第一个
func getBestEmail(emails []GitHubEmail) string {
	var verified, primary string
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email // 最佳选择
		}
		if e.Verified && verified == "" {
			verified = e.Email
		}
		if e.Primary && primary == "" {
			primary = e.Email
		}
	}
	if verified != "" {
		return verified
	}
	if primary != "" {
		return primary
	}
	if len(emails) > 0 {
		return emails[0].Email
	}
	return ""
}
```

- [ ] **Step 2: 编写 getBestEmail 单元测试**

在 `internal/handler/oauth_handler_test.go` 中添加测试：

```go
func TestGetBestEmail(t *testing.T) {
	tests := []struct {
		name     string
		emails   []GitHubEmail
		expected string
	}{
		{
			name:     "empty emails",
			emails:   []GitHubEmail{},
			expected: "",
		},
		{
			name: "primary and verified is best",
			emails: []GitHubEmail{
				{Email: "a@example.com", Primary: false, Verified: true},
				{Email: "b@example.com", Primary: true, Verified: true},
				{Email: "c@example.com", Primary: true, Verified: false},
			},
			expected: "b@example.com",
		},
		{
			name: "verified without primary",
			emails: []GitHubEmail{
				{Email: "a@example.com", Primary: false, Verified: false},
				{Email: "b@example.com", Primary: false, Verified: true},
			},
			expected: "b@example.com",
		},
		{
			name: "primary without verified",
			emails: []GitHubEmail{
				{Email: "a@example.com", Primary: false, Verified: true},
				{Email: "b@example.com", Primary: true, Verified: false},
			},
			expected: "a@example.com", // verified 优先级高于 primary
		},
		{
			name: "first email as fallback",
			emails: []GitHubEmail{
				{Email: "a@example.com", Primary: false, Verified: false},
				{Email: "b@example.com", Primary: false, Verified: false},
			},
			expected: "a@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBestEmail(tt.emails)
			assert.Equal(t, tt.expected, result)
		})
	}
}
```

- [ ] **Step 3: 运行测试验证通过**

Run: `go test ./internal/handler/... -run TestGetBestEmail -v`
Expected: PASS

- [ ] **Step 4: 提交辅助函数**

```bash
git add internal/handler/oauth_handler.go internal/handler/oauth_handler_test.go
git commit -m "feat(oauth): add getBestEmail helper function with tests"
```

---

## Task 5: 扩展 OAuthHandler - 添加依赖和新端点

**Files:**
- Modify: `internal/handler/oauth_handler.go`

- [ ] **Step 1: 修改 OAuthHandler 结构体添加 pendingOAuthRepo 字段**

在 `OAuthHandler` 结构体中添加新字段：

```go
type OAuthHandler struct {
	cfg              *config.Config
	jwtService       *jwt.Service
	userRepo         service.UserRepository
	tokenRepo        service.RefreshTokenRepository
	stateRepo        service.OAuthStateRepository
	pendingOAuthRepo service.PendingOAuthRepository // 新增
	githubBaseURL    string
}
```

- [ ] **Step 2: 修改 NewOAuthHandler 构造函数**

更新构造函数签名和实现：

```go
func NewOAuthHandler(
	cfg *config.Config,
	jwtService *jwt.Service,
	userRepo service.UserRepository,
	tokenRepo service.RefreshTokenRepository,
	stateRepo service.OAuthStateRepository,
	pendingOAuthRepo service.PendingOAuthRepository, // 新增
	githubBaseURL string,
) *OAuthHandler {
	return &OAuthHandler{
		cfg:              cfg,
		jwtService:       jwtService,
		userRepo:         userRepo,
		tokenRepo:        tokenRepo,
		stateRepo:        stateRepo,
		pendingOAuthRepo: pendingOAuthRepo, // 新增
		githubBaseURL:    githubBaseURL,
	}
}
```

- [ ] **Step 3: 添加 fetchGitHubEmails 方法**

添加获取 GitHub 邮箱的方法：

```go
// fetchGitHubEmails 调用 GitHub /user/emails API 获取用户邮箱列表
func (h *OAuthHandler) fetchGitHubEmails(ctx context.Context, accessToken string) ([]GitHubEmail, error) {
	url := h.githubBaseURL + "/user/emails"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var emails []GitHubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return nil, err
	}

	return emails, nil
}
```

- [ ] **Step 4: 添加 GetPendingOAuth 处理器**

添加获取待确认 OAuth 信息的处理器：

```go
// GetPendingOAuth 处理 GET /api/v1/auth/oauth/pending
func (h *OAuthHandler) GetPendingOAuth(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "token is required",
		})
	}

	pending, err := h.pendingOAuthRepo.GetByToken(c.Context(), token)
	if err != nil || pending.IsExpired() {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "token not found or expired",
		})
	}

	return c.JSON(fiber.Map{
		"github_login": pending.GitHubLogin,
		"nickname":     pending.Nickname,
		"avatar_url":   pending.AvatarURL,
	})
}
```

- [ ] **Step 5: 添加 CompleteOAuth 处理器**

添加完成 OAuth 注册的处理器：

```go
// CompleteOAuthRequest 完成注册请求体
type CompleteOAuthRequest struct {
	Token string `json:"token"`
	Email string `json:"email"`
}

// CompleteOAuth 处理 POST /api/v1/auth/oauth/complete
func (h *OAuthHandler) CompleteOAuth(c *fiber.Ctx) error {
	var req CompleteOAuthRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// 验证 token
	if req.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "token is required",
		})
	}

	// 验证邮箱格式
	if req.Email == "" || !isValidEmail(req.Email) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "valid email is required",
		})
	}

	// 获取待确认数据
	pending, err := h.pendingOAuthRepo.GetByToken(c.Context(), req.Token)
	if err != nil || pending.IsExpired() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "token not found or expired",
		})
	}

	// 检查邮箱是否已被使用
	existingUser, _ := h.userRepo.GetByEmail(c.Context(), req.Email)
	if existingUser != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "EMAIL_ALREADY_USED",
				"message": "该邮箱已被注册，请使用其他邮箱",
			},
		})
	}

	// 创建新用户
	user := &model.User{
		Email:        req.Email,
		Nickname:     pending.Nickname,
		AvatarURL:    pending.AvatarURL,
		AuthProvider: "github",
		GitHubID:     pending.GitHubID,
		GitHubLogin:  pending.GitHubLogin,
	}

	if err := h.userRepo.Create(c.Context(), user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create user",
		})
	}

	// 删除 pending 记录
	_ = h.pendingOAuthRepo.Delete(c.Context(), req.Token)

	// 生成令牌并设置 cookie
	accessToken, refreshToken, err := h.generateTokens(c.Context(), user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to generate tokens",
		})
	}

	h.setAuthCookies(c, accessToken, refreshToken)

	return c.JSON(fiber.Map{
		"id":           user.ID,
		"email":        user.Email,
		"nickname":     user.Nickname,
		"avatar_url":   user.AvatarURL,
		"auth_provider": user.AuthProvider,
	})
}

// isValidEmail 验证邮箱格式
func isValidEmail(email string) bool {
	// 简单的邮箱格式验证
	if len(email) > 255 {
		return false
	}
	// 基本格式检查
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}
```

- [ ] **Step 6: 验证编译通过**

Run: `go build ./internal/handler/...`
Expected: 编译成功

- [ ] **Step 7: 提交 Handler 扩展**

```bash
git add internal/handler/oauth_handler.go
git commit -m "feat(oauth): add pendingOAuthRepo dependency and new endpoints"
```

---

## Task 6: 修改 GitHub Callback 流程

**Files:**
- Modify: `internal/handler/oauth_handler.go`

- [ ] **Step 1: 修改 HandleCallback 方法**

修改现有的 `HandleCallback` 方法，在获取 GitHub 用户信息后添加邮箱获取逻辑：

找到 `HandleCallback` 方法中获取 `githubUser` 之后、查找用户之前的位置，添加以下逻辑：

```go
// 在获取 githubUser 之后，查找用户之前添加:

// 如果用户没有公开邮箱，尝试从 /user/emails 获取
	email := githubUser.Email
	if email == "" {
		emails, err := h.fetchGitHubEmails(c.Context(), accessToken)
		if err == nil && len(emails) > 0 {
			email = getBestEmail(emails)
		}
	}

	// 查找现有用户
	user, err := h.userRepo.GetByGitHubID(c.Context(), githubUser.ID)
	if err == nil && user != nil {
		// 更新 GitHubLogin（如果变化）
		if user.GitHubLogin != githubUser.Login {
			user.GitHubLogin = githubUser.Login
			_ = h.userRepo.Update(c.Context(), user)
		}
		// 直接登录
		return h.loginUser(c, user)
	}

	// 未通过 GitHub ID 找到，检查邮箱
	if email != "" {
		user, err := h.userRepo.GetByEmail(c.Context(), email)
		if err == nil && user != nil {
			// 自动关联 GitHub 账户
			user.GitHubID = githubUser.ID
			user.GitHubLogin = githubUser.Login
			if user.AvatarURL == "" {
				user.AvatarURL = githubUser.AvatarURL
			}
			_ = h.userRepo.Update(c.Context(), user)
			return h.loginUser(c, user)
		}

		// 创建新用户
		newUser := &model.User{
			Email:        email,
			Nickname:     getNickname(githubUser),
			AvatarURL:    githubUser.AvatarURL,
			AuthProvider: "github",
			GitHubID:     githubUser.ID,
			GitHubLogin:  githubUser.Login,
		}
		if err := h.userRepo.Create(c.Context(), newUser); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create user"})
		}
		return h.loginUser(c, newUser)
	}

	// 无邮箱，创建 PendingOAuth
	state, err := generateState()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate token"})
	}

	pending := &model.PendingOAuth{
		Token:       state,
		GitHubID:    githubUser.ID,
		GitHubLogin: githubUser.Login,
		Nickname:    getNickname(githubUser),
		AvatarURL:   githubUser.AvatarURL,
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	}
	if err := h.pendingOAuthRepo.Create(c.Context(), pending); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create pending oauth"})
	}

	// 重定向到前端待确认页面
	callbackHost := os.Getenv("GITHUB_CALLBACK_HOST")
	if callbackHost == "" {
		callbackHost = h.cfg.FrontendURL
	}
	return c.Redirect(callbackHost + "/oauth/pending?token=" + state)
```

- [ ] **Step 2: 添加辅助方法**

添加 `loginUser` 辅助方法（如果不存在）：

```go
// loginUser 执行用户登录逻辑
func (h *OAuthHandler) loginUser(c *fiber.Ctx, user *model.User) error {
	accessToken, refreshToken, err := h.generateTokens(c.Context(), user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to generate tokens",
		})
	}

	h.setAuthCookies(c, accessToken, refreshToken)

	// 重定向到前端
	callbackHost := os.Getenv("GITHUB_CALLBACK_HOST")
	if callbackHost == "" {
		callbackHost = h.cfg.FrontendURL
	}
	return c.Redirect(callbackHost + "/")
}

// getNickname 获取用户昵称
func getNickname(user *GitHubUser) string {
	if user.Name != "" {
		return user.Name
	}
	return user.Login
}
```

- [ ] **Step 3: 添加 strings 和 os 导入（如果需要）**

确保文件顶部有必要的导入：

```go
import (
	// ... 现有导入
	"os"
	"strings"
)
```

- [ ] **Step 4: 验证编译通过**

Run: `go build ./internal/handler/...`
Expected: 编译成功

- [ ] **Step 5: 提交 Callback 流程修改**

```bash
git add internal/handler/oauth_handler.go
git commit -m "feat(oauth): enhance callback with email fetch and pending flow"
```

---

## Task 7: 更新 main.go 依赖注入和路由

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: 初始化 PendingOAuthRepository**

在 `main.go` 中找到 repository 初始化部分，添加：

```go
// 在其他 repository 初始化之后
pendingOAuthRepo := repository.NewPendingOAuthRepository(db)
```

- [ ] **Step 2: 更新 OAuthHandler 初始化**

修改 `NewOAuthHandler` 调用，添加新参数：

```go
oauthHandler := handler.NewOAuthHandler(
	cfg,
	jwtService,
	userRepo,
	tokenRepo,
	stateRepo,
	pendingOAuthRepo, // 新增
	githubBaseURL,
)
```

- [ ] **Step 3: 添加新路由**

在路由注册部分添加新端点：

```go
// 在 OAuth 路由组中添加
oauth := api.Group("/auth/oauth")
oauth.Get("/pending", oauthHandler.GetPendingOAuth)
oauth.Post("/complete", oauthHandler.CompleteOAuth)
```

- [ ] **Step 4: 验证编译和运行**

Run: `go build ./cmd/server/...`
Expected: 编译成功

- [ ] **Step 5: 提交 main.go 变更**

```bash
git add cmd/server/main.go
git commit -m "feat(server): wire up PendingOAuthRepository and new routes"
```

---

## Task 8: 添加 Handler 集成测试

**Files:**
- Modify: `internal/handler/oauth_handler_test.go`

- [ ] **Step 1: 添加 PendingOAuth mock 测试设置**

在测试文件中添加新的 mock 服务器支持：

```go
func mockGitHubEmailsServer(t *testing.T, emails []GitHubEmail) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/emails" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(emails)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}
```

- [ ] **Step 2: 添加无邮箱用户流程测试**

```go
func TestHandleCallback_NoEmail_CreatesPendingOAuth(t *testing.T) {
	db := setupOAuthTestDB(t)
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewRefreshTokenRepository(db)
	stateRepo := repository.NewOAuthStateRepository(db)
	pendingOAuthRepo := repository.NewPendingOAuthRepository(db)

	// Mock GitHub 服务器 - 用户无公开邮箱
	githubServer := mockGitHubServerWithUser(t, GitHubUser{
		ID:        "99999",
		Login:     "noemailuser",
		Name:      "No Email User",
		Email:     "", // 无公开邮箱
		AvatarURL: "https://example.com/avatar.png",
	})
	defer githubServer.Close()

	// Mock emails API - 返回空列表
	emailsServer := mockGitHubEmailsServer(t, []GitHubEmail{})
	defer emailsServer.Close()

	jwtService := jwt.NewService("test-secret-key-32-bytes-long-1234567890")
	handler := NewOAuthHandler(
		setupOAuthTestConfig(),
		jwtService,
		userRepo,
		tokenRepo,
		stateRepo,
		pendingOAuthRepo,
		githubServer.URL,
	)

	app := fiber.New()
	app.Get("/auth/github/callback", handler.HandleCallback)

	// 创建 state
	state, _ := generateState()
	stateRepo.Create(context.Background(), &model.OAuthState{
		State:     state,
		Provider:  "github",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	})

	// 执行请求
	req := httptest.NewRequest("GET", "/auth/github/callback?code=test-code&state="+state, nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// 验证重定向到 pending 页面
	assert.Equal(t, fiber.StatusFound, resp.StatusCode)
	location := resp.Header.Get("Location")
	assert.Contains(t, location, "/oauth/pending?token=")

	// 验证 PendingOAuth 已创建
	token := strings.TrimPrefix(location, "http://example.com/oauth/pending?token=")
	pending, err := pendingOAuthRepo.GetByToken(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, "99999", pending.GitHubID)
	assert.Equal(t, "noemailuser", pending.GitHubLogin)
}
```

- [ ] **Step 3: 添加 /user/emails 获取邮箱测试**

```go
func TestHandleCallback_FetchesEmailFromAPI(t *testing.T) {
	db := setupOAuthTestDB(t)
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewRefreshTokenRepository(db)
	stateRepo := repository.NewOAuthStateRepository(db)
	pendingOAuthRepo := repository.NewPendingOAuthRepository(db)

	// Mock GitHub 服务器 - 用户无公开邮箱
	githubServer := mockGitHubServerWithUser(t, GitHubUser{
		ID:    "88888",
		Login: "privateemail",
		Name:  "Private Email",
		Email: "", // 无公开邮箱
	})
	defer githubServer.Close()

	// Mock emails API - 返回验证过的邮箱
	emailsServer := mockGitHubEmailsServer(t, []GitHubEmail{
		{Email: "verified@example.com", Primary: true, Verified: true},
	})
	defer emailsServer.Close()

	jwtService := jwt.NewService("test-secret-key-32-bytes-long-1234567890")
	handler := NewOAuthHandler(
		setupOAuthTestConfig(),
		jwtService,
		userRepo,
		tokenRepo,
		stateRepo,
		pendingOAuthRepo,
		githubServer.URL,
	)

	app := fiber.New()
	app.Get("/auth/github/callback", handler.HandleCallback)

	state, _ := generateState()
	stateRepo.Create(context.Background(), &model.OAuthState{
		State:     state,
		Provider:  "github",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	})

	req := httptest.NewRequest("GET", "/auth/github/callback?code=test-code&state="+state, nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// 验证直接登录成功
	assert.Equal(t, fiber.StatusFound, resp.StatusCode)
	location := resp.Header.Get("Location")
	assert.Contains(t, location, "/")

	// 验证用户已创建，邮箱正确
	user, _ := userRepo.GetByGitHubID(context.Background(), "88888")
	require.NotNil(t, user)
	assert.Equal(t, "verified@example.com", user.Email)
}
```

- [ ] **Step 4: 运行测试验证**

Run: `go test ./internal/handler/... -v -run "TestHandleCallback"`
Expected: PASS - 所有测试通过

- [ ] **Step 5: 提交测试**

```bash
git add internal/handler/oauth_handler_test.go
git commit -m "test(oauth): add integration tests for email fetch and pending flow"
```

---

## Task 9: 创建前端 OAuthPendingPage 组件

**Files:**
- Create: `web/src/pages/oauth/OAuthPendingPage.tsx`

- [ ] **Step 1: 创建 OAuthPendingPage 组件**

创建文件 `web/src/pages/oauth/OAuthPendingPage.tsx`：

```tsx
import { useState, useEffect } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { api } from '../../lib/api';

const emailSchema = z.object({
  email: z
    .string()
    .min(1, '请输入邮箱地址')
    .email('请输入有效的邮箱地址')
    .max(255, '邮箱地址不能超过255个字符'),
});

type EmailFormData = z.infer<typeof emailSchema>;

interface PendingOAuthInfo {
  github_login: string;
  nickname: string;
  avatar_url: string;
}

export function OAuthPendingPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [pendingInfo, setPendingInfo] = useState<PendingOAuthInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<EmailFormData>({
    resolver: zodResolver(emailSchema),
  });

  const token = searchParams.get('token');

  useEffect(() => {
    if (!token) {
      setError('无效的链接');
      setLoading(false);
      return;
    }

    const fetchPendingInfo = async () => {
      try {
        const response = await api.get<PendingOAuthInfo>(
          `/auth/oauth/pending?token=${token}`
        );
        setPendingInfo(response.data);
      } catch {
        setError('链接已过期或无效，请重新登录');
      } finally {
        setLoading(false);
      }
    };

    fetchPendingInfo();
  }, [token]);

  const onSubmit = async (data: EmailFormData) => {
    if (!token) return;

    setSubmitting(true);
    setError(null);

    try {
      await api.post('/auth/oauth/complete', {
        token,
        email: data.email,
      });
      navigate('/items');
    } catch (err: any) {
      const errorCode = err.response?.data?.error?.code;
      if (errorCode === 'EMAIL_ALREADY_USED') {
        setError('该邮箱已被注册，请使用其他邮箱');
      } else {
        setError('注册失败，请稍后重试');
      }
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-lg text-gray-600">加载中...</div>
      </div>
    );
  }

  if (error && !pendingInfo) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-center">
          <p className="text-lg text-red-600">{error}</p>
          <button
            onClick={() => navigate('/login')}
            className="mt-4 rounded-lg bg-blue-600 px-4 py-2 text-white hover:bg-blue-700"
          >
            返回登录
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 px-4">
      <div className="w-full max-w-md rounded-lg bg-white p-8 shadow-lg">
        <div className="mb-6 flex items-center justify-center gap-4">
          {pendingInfo?.avatar_url && (
            <img
              src={pendingInfo.avatar_url}
              alt="GitHub Avatar"
              className="h-16 w-16 rounded-full"
            />
          )}
          <div>
            <h1 className="text-2xl font-bold text-gray-900">
              欢迎来到 oReader！
            </h1>
          </div>
        </div>

        <div className="mb-6 rounded-lg bg-gray-50 p-4">
          <p className="text-sm text-gray-600">
            <strong>GitHub 账户:</strong> @{pendingInfo?.github_login}
          </p>
          {pendingInfo?.nickname && (
            <p className="mt-1 text-sm text-gray-600">
              <strong>昵称:</strong> {pendingInfo.nickname}
            </p>
          )}
        </div>

        <form onSubmit={handleSubmit(onSubmit)}>
          <div className="mb-4">
            <label
              htmlFor="email"
              className="mb-2 block text-sm font-medium text-gray-700"
            >
              请提供您的邮箱地址：<span className="text-red-500">*</span>
            </label>
            <input
              id="email"
              type="email"
              {...register('email')}
              className="w-full rounded-lg border border-gray-300 px-4 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              placeholder="your@email.com"
            />
            {errors.email && (
              <p className="mt-1 text-sm text-red-600">{errors.email.message}</p>
            )}
          </div>

          {error && (
            <div className="mb-4 rounded-lg bg-red-50 p-3 text-sm text-red-600">
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={submitting}
            className="w-full rounded-lg bg-blue-600 py-3 font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {submitting ? '处理中...' : '完成注册'}
          </button>
        </form>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: 验证 TypeScript 编译**

Run: `cd web && npm run typecheck`
Expected: 编译成功，无类型错误

- [ ] **Step 3: 提交前端组件**

```bash
git add web/src/pages/oauth/OAuthPendingPage.tsx
git commit -m "feat(web): add OAuthPendingPage for email collection"
```

---

## Task 10: 添加前端路由

**Files:**
- Modify: `web/src/App.tsx`

- [ ] **Step 1: 导入 OAuthPendingPage 组件**

在 `App.tsx` 的导入部分添加：

```tsx
import { OAuthPendingPage } from './pages/oauth/OAuthPendingPage';
```

- [ ] **Step 2: 添加 /oauth/pending 路由**

在路由配置中添加新路由（在公开路由区域，`/login` 附近）：

```tsx
<Route
  path="/oauth/pending"
  element={isAuthenticated ? <Navigate to="/items" /> : <OAuthPendingPage />}
/>
```

- [ ] **Step 3: 验证前端编译和运行**

Run: `cd web && npm run build`
Expected: 编译成功

- [ ] **Step 4: 提交路由变更**

```bash
git add web/src/App.tsx
git commit -m "feat(web): add /oauth/pending route"
```

---

## Task 11: 数据库迁移和最终验证

- [ ] **Step 1: 运行数据库迁移（如果需要）**

确认 GORM AutoMigrate 会自动处理新表和字段：

```go
// 在 main.go 中确认 AutoMigrate 包含新模型
db.AutoMigrate(
    &model.User{},
    &model.OAuthState{},
    &model.RefreshToken{},
    &model.PendingOAuth{}, // 新增
)
```

- [ ] **Step 2: 运行所有后端测试**

Run: `go test ./... -v`
Expected: PASS - 所有测试通过

- [ ] **Step 3: 运行前端类型检查**

Run: `cd web && npm run typecheck`
Expected: 无类型错误

- [ ] **Step 4: 手动集成测试**

1. 启动后端服务
2. 启动前端服务
3. 使用 GitHub 账户登录（测试有邮箱和无邮箱两种情况）
4. 验证无邮箱用户看到邮箱输入页面
5. 验证输入邮箱后成功登录

- [ ] **Step 5: 最终提交（如果有遗漏的文件）**

```bash
git status
# 添加任何未跟踪的文件
git add .
git commit -m "chore: final cleanup for GitHub OAuth auto-create user feature"
```

---

## 完成检查清单

- [ ] 所有测试通过 (`go test ./...`)
- [ ] 前端类型检查通过 (`cd web && npm run typecheck`)
- [ ] 数据库迁移正常（新表和字段创建成功）
- [ ] 无邮箱 GitHub 用户可以完成注册流程
- [ ] 有邮箱 GitHub 用户直接登录成功
- [ ] 邮箱匹配现有用户时自动关联成功
- [ ] 5分钟过期机制正常工作
