# PDF 论文导入功能实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 oReader 增加 PDF 论文导入功能，支持上传学术论文 PDF，通过 MinerU + LLM API 转换为高质量 Markdown 并提取元数据，在 Web 页面上展示和管理。

**Architecture:** 三层架构 — Go 后端处理文件上传和异步任务调度；Python gRPC 服务负责 MinerU PDF 解析和 LLM 精修；React 前端提供论文上传/浏览/详情展示页面。Go 后端通过 gRPC 调用 Python 服务，使用流式进度推送。

**Tech Stack:** Go (Gin + GORM) / Python (gRPC + MinerU + OpenAI API) / React (TanStack Query + Zustand + Tailwind) / Protobuf

---

## 文件结构

### Go 后端新增文件

| 文件 | 职责 |
|------|------|
| `internal/model/paper.go` | Paper/PaperTag/PaperCollection/PaperCollectionItem 模型定义 |
| `internal/repository/paper_repository.go` | 论文数据访问层 (CRUD + 搜索筛选) |
| `internal/service/interfaces.go` (修改) | 新增 PaperService/PaperRepository 接口 |
| `internal/service/paper_service.go` | 论文业务逻辑 (上传/转换调度/元数据更新) |
| `internal/handler/paper_handler.go` | 论文 HTTP API handler |
| `internal/infra/grpc/paper_client.go` | gRPC client 封装 |
| `converter/proto/paper.proto` | protobuf 定义 |
| `converter/proto/paper_grpc.pb.go` | protobuf 生成代码 |
| `converter/proto/paper.pb.go` | protobuf 生成代码 |

### Python gRPC 服务新增文件

| 文件 | 职责 |
|------|------|
| `converter/server.py` | gRPC 服务入口 |
| `converter/converter.py` | MinerU + LLM 转换逻辑 |
| `converter/requirements.txt` | Python 依赖 |
| `converter/tests/__init__.py` | 测试包 |
| `converter/tests/test_converter.py` | 转换逻辑单元测试 |

### 前端新增文件

| 文件 | 职责 |
|------|------|
| `web/src/types/paper.ts` | 论文相关 TypeScript 类型 |
| `web/src/hooks/usePapers.ts` | 论文 API hooks (TanStack Query) |
| `web/src/stores/papersStore.ts` | 论文客户端状态管理 (Zustand) |
| `web/src/components/papers/PaperList.tsx` | 论文列表组件 |
| `web/src/components/papers/PaperUpload.tsx` | 上传模态框组件 |
| `web/src/components/papers/PaperMeta.tsx` | 论文元数据头部组件 |
| `web/src/pages/papers/PapersPage.tsx` | 论文列表页 |
| `web/src/pages/papers/PaperViewPage.tsx` | 论文详情页 |

### 修改现有文件

| 文件 | 修改内容 |
|------|----------|
| `cmd/server/main.go` | 注册 paper 路由、初始化 repository/service/handler、AutoMigrate |
| `internal/config/config.go` | 新增 PaperConfig (gRPC 地址、上传目录等) |
| `.env.example` | 新增 PAPER_* 环境变量 |
| `web/src/App.tsx` | 新增 /papers 路由 |
| `web/src/components/feed/Sidebar.tsx` | 侧边栏新增论文库入口 |

---

## Task 1: Protobuf 定义

**Files:**
- Create: `converter/proto/paper.proto`

- [ ] **Step 1: 创建 protobuf 定义文件**

```protobuf
syntax = "proto3";

package paper;

option go_package = "oreader/converter/proto";

service PaperConverter {
  // 将 PDF 转换为 Markdown (流式进度)
  rpc Convert(PdfConvertRequest) returns (stream ConvertProgress);
  // 从 Markdown 提取论文元数据
  rpc ExtractMetadata(MetadataRequest) returns (PaperMetadata);
}

message PdfConvertRequest {
  bytes pdf_content = 1;
  string filename = 2;
}

message ConvertProgress {
  string status = 1;    // mining / llm_refining / completed / failed
  int32 progress = 2;   // 0-100
  string markdown = 3;  // 最终结果 (status=completed 时填充)
  string error = 4;
}

message MetadataRequest {
  string markdown = 1;
}

message PaperMetadata {
  string title = 1;
  repeated string authors = 2;
  string abstract = 3;
  repeated string keywords = 4;
  string published_year = 5;
  string doi = 6;
}
```

- [ ] **Step 2: 安装 protoc 工具并生成 Go 代码**

Run: `cd /data00/home/wangyang.backend/work/oReader && go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest`

Run: `protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative converter/proto/paper.proto`

Expected: 生成 `converter/proto/paper.pb.go` 和 `converter/proto/paper_grpc.pb.go`

- [ ] **Step 3: 初始化 Python protobuf 生成**

Run: `pip install grpcio grpcio-tools`

Run: `python -m grpc_tools.protoc -I converter/proto --python_out=converter/proto --grpc_python_out=converter/proto converter/proto/paper.proto`

Expected: 生成 `converter/proto/paper_pb2.py` 和 `converter/proto/paper_pb2_grpc.py`

- [ ] **Step 4: 添加 go 依赖**

Run: `cd /data00/home/wangyang.backend/work/oReader && go get google.golang.org/grpc google.golang.org/protobuf`

- [ ] **Step 5: Commit**

```bash
git add converter/proto/ go.mod go.sum
git commit -m "feat(papers): add protobuf definitions for paper converter gRPC service"
```

---

## Task 2: Go 数据模型

**Files:**
- Create: `internal/model/paper.go`
- Test: `internal/model/paper_test.go`

- [ ] **Step 1: 写失败测试**

```go
// internal/model/paper_test.go
package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaperStatusConstants(t *testing.T) {
	assert.Equal(t, "pending", PaperStatusPending)
	assert.Equal(t, "processing", PaperStatusProcessing)
	assert.Equal(t, "completed", PaperStatusCompleted)
	assert.Equal(t, "failed", PaperStatusFailed)
}

func TestPaper_GenerateID(t *testing.T) {
	paper := &Paper{}
	err := paper.GenerateID()
	require.NoError(t, err)
	assert.NotEmpty(t, paper.ID)
	assert.Len(t, paper.ID, 36) // UUID v7 format
}

func TestPaperCollection_GenerateID(t *testing.T) {
	collection := &PaperCollection{}
	err := collection.GenerateID()
	require.NoError(t, err)
	assert.NotEmpty(t, collection.ID)
}

func TestPaper_AuthorsJSON(t *testing.T) {
	paper := &Paper{
		Authors: `["Alice","Bob"]`,
	}
	var authors []string
	err := json.Unmarshal([]byte(paper.Authors), &authors)
	require.NoError(t, err)
	assert.Equal(t, []string{"Alice", "Bob"}, authors)
}

func TestPaper_KeywordsJSON(t *testing.T) {
	paper := &Paper{
		Keywords: `["deep learning","transformer"]`,
	}
	var keywords []string
	err := json.Unmarshal([]byte(paper.Keywords), &keywords)
	require.NoError(t, err)
	assert.Equal(t, []string{"deep learning", "transformer"}, keywords)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/model/ -run TestPaper -v`
Expected: FAIL — 编译错误，Paper 类型不存在

- [ ] **Step 3: 实现 Paper 模型**

```go
// internal/model/paper.go
package model

// Paper statuses
const (
	PaperStatusPending    = "pending"
	PaperStatusProcessing = "processing"
	PaperStatusCompleted  = "completed"
	PaperStatusFailed     = "failed"
)

// Paper represents an uploaded academic paper
type Paper struct {
	Base
	UserID           string `gorm:"type:varchar(36);not null;index" json:"user_id"`
	Title            string `gorm:"type:varchar(500)" json:"title"`
	Authors          string `gorm:"type:json" json:"authors"`
	Abstract         string `gorm:"type:text" json:"abstract"`
	Keywords         string `gorm:"type:json" json:"keywords"`
	PublishedYear    string `gorm:"type:varchar(10)" json:"published_year"`
	DOI              string `gorm:"type:varchar(200)" json:"doi"`
	PDFPath          string `gorm:"type:varchar(500)" json:"pdf_path"`
	PDFSize          int64  `json:"pdf_size"`
	MarkdownContent  string `gorm:"type:longtext" json:"markdown_content"`
	CoverImage       string `gorm:"type:varchar(500)" json:"cover_image"`
	OriginalFilename string `gorm:"type:varchar(255)" json:"original_filename"`
	Status           string `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	Error            string `gorm:"type:text" json:"error,omitempty"`
	User             *User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// PaperTag represents a tag for a paper
type PaperTag struct {
	Base
	PaperID string `gorm:"type:varchar(36);not null;index" json:"paper_id"`
	Tag     string `gorm:"type:varchar(100);not null;index" json:"tag"`
	Paper   *Paper `gorm:"foreignKey:PaperID" json:"paper,omitempty"`
}

// PaperCollection represents a user-defined paper collection
type PaperCollection struct {
	Base
	UserID      string `gorm:"type:varchar(36);not null;index" json:"user_id"`
	Name        string `gorm:"type:varchar(100);not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	User        *User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// PaperCollectionItem represents a paper in a collection
type PaperCollectionItem struct {
	Base
	CollectionID string           `gorm:"type:varchar(36);not null;index" json:"collection_id"`
	PaperID      string           `gorm:"type:varchar(36);not null;index" json:"paper_id"`
	Collection   *PaperCollection `gorm:"foreignKey:CollectionID" json:"collection,omitempty"`
	Paper        *Paper           `gorm:"foreignKey:PaperID" json:"paper,omitempty"`
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/model/ -run TestPaper -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/model/paper.go internal/model/paper_test.go
git commit -m "feat(papers): add Paper, PaperTag, PaperCollection models"
```

---

## Task 3: 配置扩展

**Files:**
- Modify: `internal/config/config.go`
- Modify: `.env.example`
- Test: `internal/config/config_test.go` (追加)

- [ ] **Step 1: 写失败测试 — 验证 PaperConfig 加载**

在 `internal/config/config_test.go` 末尾追加：

```go
func TestPaperConfigDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "test.db")
	t.Setenv("JWT_SECRET_KEY", "super-secret-key-at-least-32-characters!!")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "localhost:50051", cfg.Paper.GRPCAddr)
	assert.Equal(t, "uploads/papers", cfg.Paper.UploadDir)
	assert.Equal(t, int64(50*1024*1024), cfg.Paper.MaxUploadSize)
	assert.Equal(t, "5m", cfg.Paper.GRPCTimeout)
}

func TestPaperConfigFromEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "test.db")
	t.Setenv("JWT_SECRET_KEY", "super-secret-key-at-least-32-characters!!")
	t.Setenv("PAPER_GRPC_ADDR", "paper-service:50051")
	t.Setenv("PAPER_UPLOAD_DIR", "/data/papers")
	t.Setenv("PAPER_MAX_UPLOAD_SIZE", "104857600")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "paper-service:50051", cfg.Paper.GRPCAddr)
	assert.Equal(t, "/data/papers", cfg.Paper.UploadDir)
	assert.Equal(t, int64(104857600), cfg.Paper.MaxUploadSize)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/config/ -run TestPaperConfig -v`
Expected: FAIL — `cfg.Paper` 不存在

- [ ] **Step 3: 添加 PaperConfig**

在 `internal/config/config.go` 的 `Config` 结构体中添加：

```go
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Auth      AuthConfig
	Refresh   RefreshConfig
	RateLimit RateLimitConfig
	Logging   LoggingConfig
	OAuth     OAuthConfig
	Paper     PaperConfig
}
```

新增 PaperConfig 类型：

```go
// PaperConfig holds paper import configuration
type PaperConfig struct {
	GRPCAddr      string
	UploadDir     string
	MaxUploadSize int64
	GRPCTimeout   string
}
```

在 `Load()` 函数中添加绑定：

```go
	// Paper defaults
	v.SetDefault("PAPER_GRPC_ADDR", "localhost:50051")
	v.SetDefault("PAPER_UPLOAD_DIR", "uploads/papers")
	v.SetDefault("PAPER_MAX_UPLOAD_SIZE", 52428800) // 50MB
	v.SetDefault("PAPER_GRPC_TIMEOUT", "5m")

	_ = v.BindEnv("PAPER_GRPC_ADDR")
	_ = v.BindEnv("PAPER_UPLOAD_DIR")
	_ = v.BindEnv("PAPER_MAX_UPLOAD_SIZE")
	_ = v.BindEnv("PAPER_GRPC_TIMEOUT")
```

在 cfg 初始化中添加：

```go
	Paper: PaperConfig{
		GRPCAddr:      v.GetString("PAPER_GRPC_ADDR"),
		UploadDir:     v.GetString("PAPER_UPLOAD_DIR"),
		MaxUploadSize: v.GetInt64("PAPER_MAX_UPLOAD_SIZE"),
		GRPCTimeout:   v.GetString("PAPER_GRPC_TIMEOUT"),
	},
```

添加辅助方法：

```go
// GetGRPCTimeout returns the gRPC timeout as a duration
func (c *Config) GetGRPCTimeout() (time.Duration, error) {
	return time.ParseDuration(c.Paper.GRPCTimeout)
}
```

- [ ] **Step 4: 更新 .env.example**

在文件末尾追加：

```env
# =============================================================================
# Paper Import Configuration (Optional)
# =============================================================================

# Paper converter gRPC service address
PAPER_GRPC_ADDR=localhost:50051

# Directory for uploaded PDF files
PAPER_UPLOAD_DIR=uploads/papers

# Maximum PDF upload size in bytes (default: 52428800 = 50MB)
PAPER_MAX_UPLOAD_SIZE=52428800

# gRPC client timeout (default: 5m)
PAPER_GRPC_TIMEOUT=5m
```

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./internal/config/ -run TestPaperConfig -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go .env.example
git commit -m "feat(papers): add PaperConfig for gRPC and upload settings"
```

---

## Task 4: Paper Repository

**Files:**
- Create: `internal/repository/paper_repository.go`
- Test: `internal/repository/paper_repository_test.go`
- Modify: `internal/service/interfaces.go` (新增 PaperRepository 接口)

- [ ] **Step 1: 在 interfaces.go 中添加 PaperRepository 接口**

在 `internal/service/interfaces.go` 末尾追加：

```go
// PaperRepository defines the interface for paper data access
type PaperRepository interface {
	Create(ctx context.Context, paper *model.Paper) error
	GetByID(ctx context.Context, id string) (*model.Paper, error)
	ListByUserID(ctx context.Context, userID string, opts PaperListOptions) ([]*model.Paper, int64, error)
	Update(ctx context.Context, paper *model.Paper) error
	Delete(ctx context.Context, id string) error
	ListTags(ctx context.Context, userID string) ([]string, error)
}

// PaperTagRepository defines the interface for paper tag data access
type PaperTagRepository interface {
	SetTags(ctx context.Context, paperID string, tags []string) error
	GetByPaperID(ctx context.Context, paperID string) ([]*model.PaperTag, error)
}

// PaperCollectionRepository defines the interface for paper collection data access
type PaperCollectionRepository interface {
	Create(ctx context.Context, collection *model.PaperCollection) error
	GetByID(ctx context.Context, id string) (*model.PaperCollection, error)
	ListByUserID(ctx context.Context, userID string) ([]*model.PaperCollection, error)
	Update(ctx context.Context, collection *model.PaperCollection) error
	Delete(ctx context.Context, id string) error
	AddPaper(ctx context.Context, collectionID, paperID string) error
	RemovePaper(ctx context.Context, collectionID, paperID string) error
	ListPapers(ctx context.Context, collectionID string) ([]*model.Paper, error)
}

// PaperListOptions defines pagination and filtering for papers
type PaperListOptions struct {
	Limit  int
	Offset int
	Query  string // search title, authors, keywords
	Year   string
	Tag    string
	Status string
	Sort   string // "created_at", "title", "published_year"
	Order  string // "asc", "desc"
}
```

- [ ] **Step 2: 写失败测试**

```go
// internal/repository/paper_repository_test.go
package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"oreader/internal/model"
	"oreader/internal/service"
)

func setupPaperDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&model.User{},
		&model.Paper{},
		&model.PaperTag{},
		&model.PaperCollection{},
		&model.PaperCollectionItem{},
	)
	require.NoError(t, err)

	return db
}

func createTestUser(t *testing.T, db *gorm.DB) *model.User {
	t.Helper()
	user := &model.User{Email: "test@example.com", PasswordHash: "hash"}
	require.NoError(t, user.GenerateID())
	require.NoError(t, NewUserRepository(db).Create(context.Background(), user))
	return user
}

func TestPaperRepository_CreateAndGet(t *testing.T) {
	db := setupPaperDB(t)
	repo := NewPaperRepository(db)
	user := createTestUser(t, db)
	ctx := context.Background()

	paper := &model.Paper{
		UserID:           user.ID,
		Title:            "Test Paper",
		OriginalFilename: "test.pdf",
		PDFPath:          "uploads/papers/test.pdf",
		PDFSize:          1024,
		Status:           model.PaperStatusPending,
	}
	require.NoError(t, paper.GenerateID())

	err := repo.Create(ctx, paper)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, paper.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "Test Paper", found.Title)
	assert.Equal(t, model.PaperStatusPending, found.Status)
}

func TestPaperRepository_GetByID_NotFound(t *testing.T) {
	db := setupPaperDB(t)
	repo := NewPaperRepository(db)
	ctx := context.Background()

	found, err := repo.GetByID(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestPaperRepository_ListByUserID(t *testing.T) {
	db := setupPaperDB(t)
	repo := NewPaperRepository(db)
	user := createTestUser(t, db)
	ctx := context.Background()

	// Create 3 papers
	for i := 0; i < 3; i++ {
		paper := &model.Paper{
			UserID:           user.ID,
			Title:            "Paper " + string(rune('A'+i)),
			OriginalFilename: "test.pdf",
			PDFPath:          "uploads/papers/test.pdf",
			Status:           model.PaperStatusCompleted,
		}
		require.NoError(t, paper.GenerateID())
		require.NoError(t, repo.Create(ctx, paper))
	}

	papers, total, err := repo.ListByUserID(ctx, user.ID, service.PaperListOptions{
		Limit: 10, Offset: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, papers, 3)
}

func TestPaperRepository_ListByUserID_WithSearch(t *testing.T) {
	db := setupPaperDB(t)
	repo := NewPaperRepository(db)
	user := createTestUser(t, db)
	ctx := context.Background()

	// Create papers with different titles
	for _, title := range []string{"Attention Is All You Need", "BERT: Pre-training of Deep Bidirectional Transformers"} {
		paper := &model.Paper{
			UserID:           user.ID,
			Title:            title,
			OriginalFilename: "test.pdf",
			PDFPath:          "uploads/papers/test.pdf",
			Status:           model.PaperStatusCompleted,
		}
		require.NoError(t, paper.GenerateID())
		require.NoError(t, repo.Create(ctx, paper))
	}

	papers, _, err := repo.ListByUserID(ctx, user.ID, service.PaperListOptions{
		Limit: 10, Query: "Attention",
	})
	require.NoError(t, err)
	assert.Len(t, papers, 1)
	assert.Equal(t, "Attention Is All You Need", papers[0].Title)
}

func TestPaperRepository_Update(t *testing.T) {
	db := setupPaperDB(t)
	repo := NewPaperRepository(db)
	user := createTestUser(t, db)
	ctx := context.Background()

	paper := &model.Paper{
		UserID: user.ID,
		Status: model.PaperStatusPending,
	}
	require.NoError(t, paper.GenerateID())
	require.NoError(t, repo.Create(ctx, paper))

	paper.Title = "Updated Title"
	paper.Status = model.PaperStatusCompleted
	paper.MarkdownContent = "# Hello\n\nWorld"
	require.NoError(t, repo.Update(ctx, paper))

	found, err := repo.GetByID(ctx, paper.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", found.Title)
	assert.Equal(t, model.PaperStatusCompleted, found.Status)
}

func TestPaperRepository_Delete(t *testing.T) {
	db := setupPaperDB(t)
	repo := NewPaperRepository(db)
	user := createTestUser(t, db)
	ctx := context.Background()

	paper := &model.Paper{UserID: user.ID, Status: model.PaperStatusPending}
	require.NoError(t, paper.GenerateID())
	require.NoError(t, repo.Create(ctx, paper))

	require.NoError(t, repo.Delete(ctx, paper.ID))

	found, err := repo.GetByID(ctx, paper.ID)
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestPaperRepository_ListTags(t *testing.T) {
	db := setupPaperDB(t)
	repo := NewPaperRepository(db)
	tagRepo := NewPaperTagRepository(db)
	user := createTestUser(t, db)
	ctx := context.Background()

	paper := &model.Paper{UserID: user.ID, Status: model.PaperStatusCompleted}
	require.NoError(t, paper.GenerateID())
	require.NoError(t, repo.Create(ctx, paper))

	require.NoError(t, tagRepo.SetTags(ctx, paper.ID, []string{"deep learning", "transformer"}))

	tags, err := repo.ListTags(ctx, user.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"deep learning", "transformer"}, tags)
}
```

- [ ] **Step 3: 运行测试确认失败**

Run: `go test ./internal/repository/ -run TestPaper -v`
Expected: FAIL — 编译错误

- [ ] **Step 4: 实现 PaperRepository**

```go
// internal/repository/paper_repository.go
package repository

import (
	"context"

	"gorm.io/gorm"
	"oreader/internal/model"
	"oreader/internal/service"
)

type paperRepository struct {
	db *gorm.DB
}

// NewPaperRepository creates a new paper repository
func NewPaperRepository(db *gorm.DB) service.PaperRepository {
	return &paperRepository{db: db}
}

func (r *paperRepository) Create(ctx context.Context, paper *model.Paper) error {
	return r.db.WithContext(ctx).Create(paper).Error
}

func (r *paperRepository) GetByID(ctx context.Context, id string) (*model.Paper, error) {
	var paper model.Paper
	err := r.db.WithContext(ctx).First(&paper, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &paper, nil
}

func (r *paperRepository) ListByUserID(ctx context.Context, userID string, opts service.PaperListOptions) ([]*model.Paper, int64, error) {
	var papers []*model.Paper
	var total int64

	query := r.db.WithContext(ctx).Where("user_id = ?", userID)

	if opts.Query != "" {
		like := "%" + opts.Query + "%"
		query = query.Where("title LIKE ? OR authors LIKE ? OR keywords LIKE ?", like, like, like)
	}
	if opts.Year != "" {
		query = query.Where("published_year = ?", opts.Year)
	}
	if opts.Status != "" {
		query = query.Where("status = ?", opts.Status)
	}
	if opts.Tag != "" {
		query = query.Where("id IN (SELECT paper_id FROM paper_tags WHERE tag = ?)", opts.Tag)
	}

	// Count total
	if err := query.Model(&model.Paper{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Sort
	sort := "created_at"
	if opts.Sort != "" {
		sort = opts.Sort
	}
	order := "DESC"
	if opts.Order != "" {
		order = opts.Order
	}
	query = query.Order(sort + " " + order)

	// Paginate
	if opts.Limit > 0 {
		query = query.Limit(opts.Limit)
	}
	if opts.Offset > 0 {
		query = query.Offset(opts.Offset)
	}

	err := query.Find(&papers).Error
	return papers, total, err
}

func (r *paperRepository) Update(ctx context.Context, paper *model.Paper) error {
	return r.db.WithContext(ctx).Save(paper).Error
}

func (r *paperRepository) Delete(ctx context.Context, id string) error {
	// Delete tags first, then paper
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("paper_id = ?", id).Delete(&model.PaperTag{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Paper{}, id).Error
	})
}

func (r *paperRepository) ListTags(ctx context.Context, userID string) ([]string, error) {
	var tags []string
	err := r.db.WithContext(ctx).
		Distinct("tag").
		Joins("JOIN papers ON papers.id = paper_tags.paper_id").
		Where("papers.user_id = ?", userID).
		Pluck("paper_tags.tag", &tags).Error
	return tags, err
}
```

- [ ] **Step 5: 实现 PaperTagRepository**

```go
// 在 internal/repository/paper_repository.go 中追加

type paperTagRepository struct {
	db *gorm.DB
}

// NewPaperTagRepository creates a new paper tag repository
func NewPaperTagRepository(db *gorm.DB) service.PaperTagRepository {
	return &paperTagRepository{db: db}
}

func (r *paperTagRepository) SetTags(ctx context.Context, paperID string, tags []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete existing tags
		if err := tx.Where("paper_id = ?", paperID).Delete(&model.PaperTag{}).Error; err != nil {
			return err
		}
		// Create new tags
		for _, tag := range tags {
			pt := &model.PaperTag{PaperID: paperID, Tag: tag}
			if err := pt.GenerateID(); err != nil {
				return err
			}
			if err := tx.Create(pt).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *paperTagRepository) GetByPaperID(ctx context.Context, paperID string) ([]*model.PaperTag, error) {
	var tags []*model.PaperTag
	err := r.db.WithContext(ctx).Where("paper_id = ?", paperID).Find(&tags).Error
	return tags, err
}
```

- [ ] **Step 6: 运行测试确认通过**

Run: `go test ./internal/repository/ -run TestPaper -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/repository/paper_repository.go internal/repository/paper_repository_test.go internal/service/interfaces.go
git commit -m "feat(papers): add PaperRepository and PaperTagRepository with CRUD + search"
```

---

## Task 5: gRPC Client

**Files:**
- Create: `internal/infra/grpc/paper_client.go`
- Test: `internal/infra/grpc/paper_client_test.go`

- [ ] **Step 1: 写失败测试**

```go
// internal/infra/grpc/paper_client_test.go
package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPaperClient_InvalidAddress(t *testing.T) {
	client, err := NewPaperClient("invalid:99999", 5*time.Second)
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestPaperClientInterface(t *testing.T) {
	// Verify the interface is implemented
	var _ PaperConverterClient = (*paperClient)(nil)
}

func TestPaperClient_ContextCancelled(t *testing.T) {
	// When context is cancelled before RPC, should return error
	client, err := NewPaperClient("localhost:50051", 1*time.Second)
	if err != nil {
		// gRPC server not available, skip
		t.Skip("gRPC server not available")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = client.Convert(ctx, nil, "test.pdf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/infra/grpc/ -v`
Expected: FAIL — 包不存在

- [ ] **Step 3: 实现 gRPC Client**

```go
// internal/infra/grpc/paper_client.go
package grpc

import (
	"context"
	"io"
	"time"

	pb "oreader/converter/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ConvertProgress represents a progress update from the converter
type ConvertProgress struct {
	Status   string
	Progress int32
	Markdown string
	Error    string
}

// PaperMetadata represents extracted paper metadata
type PaperMetadata struct {
	Title         string
	Authors       []string
	Abstract      string
	Keywords      []string
	PublishedYear string
	DOI           string
}

// PaperConverterClient defines the interface for the paper converter gRPC client
type PaperConverterClient interface {
	Convert(ctx context.Context, pdfContent []byte, filename string) ([]ConvertProgress, error)
	ExtractMetadata(ctx context.Context, markdown string) (*PaperMetadata, error)
	Close() error
}

type paperClient struct {
	conn   *grpc.ClientConn
	client pb.PaperConverterClient
}

// NewPaperClient creates a new gRPC client for the paper converter service
func NewPaperClient(addr string, timeout time.Duration) (PaperConverterClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}

	return &paperClient{
		conn:   conn,
		client: pb.NewPaperConverterClient(conn),
	}, nil
}

func (c *paperClient) Convert(ctx context.Context, pdfContent []byte, filename string) ([]ConvertProgress, error) {
	stream, err := c.client.Convert(ctx, &pb.PdfConvertRequest{
		PdfContent: pdfContent,
		Filename:   filename,
	})
	if err != nil {
		return nil, err
	}

	var results []ConvertProgress
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return results, err
		}
		results = append(results, ConvertProgress{
			Status:   resp.Status,
			Progress: resp.Progress,
			Markdown: resp.Markdown,
			Error:    resp.Error,
		})
	}
	return results, nil
}

func (c *paperClient) ExtractMetadata(ctx context.Context, markdown string) (*PaperMetadata, error) {
	resp, err := c.client.ExtractMetadata(ctx, &pb.MetadataRequest{
		Markdown: markdown,
	})
	if err != nil {
		return nil, err
	}

	return &PaperMetadata{
		Title:         resp.Title,
		Authors:       resp.Authors,
		Abstract:      resp.Abstract,
		Keywords:      resp.Keywords,
		PublishedYear: resp.PublishedYear,
		DOI:           resp.Doi,
	}, nil
}

func (c *paperClient) Close() error {
	return c.conn.Close()
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/infra/grpc/ -v`
Expected: PASS (ContextCancelled 测试可能被 skip)

- [ ] **Step 5: Commit**

```bash
git add internal/infra/grpc/paper_client.go internal/infra/grpc/paper_client_test.go
git commit -m "feat(papers): add gRPC client for paper converter service"
```

---

## Task 6: Paper Service

**Files:**
- Create: `internal/service/paper_service.go`
- Test: `internal/service/paper_service_test.go`
- Modify: `internal/service/interfaces.go` (新增 PaperService 接口)

- [ ] **Step 1: 在 interfaces.go 中添加 PaperService 接口**

```go
// PaperService defines the interface for paper business logic
type PaperService interface {
	UploadPaper(ctx context.Context, userID string, filename string, pdfContent []byte) (*model.Paper, error)
	GetPaper(ctx context.Context, userID, paperID string) (*model.Paper, error)
	ListPapers(ctx context.Context, userID string, opts PaperListOptions) ([]*model.Paper, int64, error)
	UpdatePaper(ctx context.Context, userID, paperID string, updates map[string]interface{}) (*model.Paper, error)
	DeletePaper(ctx context.Context, userID, paperID string) error
	GetPaperStatus(ctx context.Context, userID, paperID string) (*PaperStatusResponse, error)
	RetryPaper(ctx context.Context, userID, paperID string) (*model.Paper, error)
	ListTags(ctx context.Context, userID string) ([]string, error)
	UpdateTags(ctx context.Context, userID, paperID string, tags []string) error
	DownloadPaper(ctx context.Context, userID, paperID string) (string, string, error)
}

// PaperStatusResponse represents the conversion status of a paper
type PaperStatusResponse struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Progress  int32  `json:"progress"`
	Error     string `json:"error,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
```

- [ ] **Step 2: 写失败测试**

```go
// internal/service/paper_service_test.go
package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"oreader/internal/infra/grpc"
	"oreader/internal/model"
)

// mockPaperConverterClient is a mock for the gRPC client
type mockPaperConverterClient struct {
	mock.Mock
}

func (m *mockPaperConverterClient) Convert(ctx context.Context, pdfContent []byte, filename string) ([]grpc.ConvertProgress, error) {
	args := m.Called(ctx, pdfContent, filename)
	if results := args.Get(0); results != nil {
		return results.([]grpc.ConvertProgress), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockPaperConverterClient) ExtractMetadata(ctx context.Context, markdown string) (*grpc.PaperMetadata, error) {
	args := m.Called(ctx, markdown)
	if results := args.Get(0); results != nil {
		return results.(*grpc.PaperMetadata), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockPaperConverterClient) Close() error {
	return nil
}

func setupPaperServiceDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&model.User{},
		&model.Paper{},
		&model.PaperTag{},
	)
	require.NoError(t, err)

	return db
}

func createPaperUser(t *testing.T, db *gorm.DB) string {
	t.Helper()
	user := &model.User{Email: "paper-test@example.com", PasswordHash: "hash"}
	require.NoError(t, user.GenerateID())
	require.NoError(t, db.Create(user).Error)
	return user.ID
}

func TestPaperService_UploadPaper(t *testing.T) {
	db := setupPaperServiceDB(t)
	userID := createPaperUser(t, db)

	tmpDir := t.TempDir()

	mockClient := new(mockPaperConverterClient)
	mockClient.On("Convert", mock.Anything, mock.Anything, "test.pdf").
		Return([]grpc.ConvertProgress{
			{Status: "mining", Progress: 50},
			{Status: "completed", Progress: 100, Markdown: "# Test Paper\n\nContent here"},
		}, nil)
	mockClient.On("ExtractMetadata", mock.Anything, "# Test Paper\n\nContent here").
		Return(&grpc.PaperMetadata{
			Title:         "Test Paper",
			Authors:       []string{"Alice", "Bob"},
			Abstract:      "This is a test paper abstract",
			Keywords:      []string{"machine learning"},
			PublishedYear: "2024",
			DOI:           "10.1234/test",
		}, nil)

	paperRepo := NewPaperRepositoryTest(db)
	svc := NewPaperService(paperRepo, nil, mockClient, tmpDir)

	paper, err := svc.UploadPaper(context.Background(), userID, "test.pdf", []byte("fake-pdf-content"))
	require.NoError(t, err)
	assert.Equal(t, "Test Paper", paper.Title)
	assert.Equal(t, model.PaperStatusCompleted, paper.Status)
	assert.Contains(t, paper.PDFPath, "test.pdf")
}

func TestPaperService_UploadPaper_GRPCFailed(t *testing.T) {
	db := setupPaperServiceDB(t)
	userID := createPaperUser(t, db)

	tmpDir := t.TempDir()

	mockClient := new(mockPaperConverterClient)
	mockClient.On("Convert", mock.Anything, mock.Anything, "test.pdf").
		Return(nil, assert.AnError)

	paperRepo := NewPaperRepositoryTest(db)
	svc := NewPaperService(paperRepo, nil, mockClient, tmpDir)

	paper, err := svc.UploadPaper(context.Background(), userID, "test.pdf", []byte("fake-pdf-content"))
	require.NoError(t, err)
	assert.Equal(t, model.PaperStatusFailed, paper.Status)
	assert.Contains(t, paper.Error, "assert.AnError")
}

func TestPaperService_GetPaper(t *testing.T) {
	db := setupPaperServiceDB(t)
	userID := createPaperUser(t, db)

	paper := &model.Paper{
		UserID: userID,
		Title:  "My Paper",
		Status: model.PaperStatusCompleted,
	}
	require.NoError(t, paper.GenerateID())
	require.NoError(t, db.Create(paper).Error)

	paperRepo := NewPaperRepositoryTest(db)
	svc := NewPaperService(paperRepo, nil, nil, t.TempDir())

	found, err := svc.GetPaper(context.Background(), userID, paper.ID)
	require.NoError(t, err)
	assert.Equal(t, "My Paper", found.Title)
}

func TestPaperService_GetPaper_NotOwner(t *testing.T) {
	db := setupPaperServiceDB(t)
	userID := createPaperUser(t, db)

	paper := &model.Paper{
		UserID: userID,
		Status: model.PaperStatusCompleted,
	}
	require.NoError(t, paper.GenerateID())
	require.NoError(t, db.Create(paper).Error)

	paperRepo := NewPaperRepositoryTest(db)
	svc := NewPaperService(paperRepo, nil, nil, t.TempDir())

	_, err := svc.GetPaper(context.Background(), "other-user-id", paper.ID)
	require.Error(t, err)
	assert.Equal(t, ErrPaperNotFound, err)
}

func TestPaperService_DeletePaper(t *testing.T) {
	db := setupPaperServiceDB(t)
	userID := createPaperUser(t, db)

	// Create paper with PDF file
	tmpDir := t.TempDir()
	pdfPath := filepath.Join(tmpDir, userID, "test.pdf")
	require.NoError(t, os.MkdirAll(filepath.Dir(pdfPath), 0755))
	require.NoError(t, os.WriteFile(pdfPath, []byte("fake-pdf"), 0644))

	paper := &model.Paper{
		UserID: userID,
		Status: model.PaperStatusCompleted,
		PDFPath: pdfPath,
	}
	require.NoError(t, paper.GenerateID())
	require.NoError(t, db.Create(paper).Error)

	paperRepo := NewPaperRepositoryTest(db)
	svc := NewPaperService(paperRepo, nil, nil, tmpDir)

	err := svc.DeletePaper(context.Background(), userID, paper.ID)
	require.NoError(t, err)

	// Verify file is deleted
	_, err = os.Stat(pdfPath)
	assert.True(t, os.IsNotExist(err))
}

func TestPaperService_UpdateTags(t *testing.T) {
	db := setupPaperServiceDB(t)
	userID := createPaperUser(t, db)

	paper := &model.Paper{UserID: userID, Status: model.PaperStatusCompleted}
	require.NoError(t, paper.GenerateID())
	require.NoError(t, db.Create(paper).Error)

	paperRepo := NewPaperRepositoryTest(db)
	tagRepo := NewPaperTagRepositoryTest(db)
	svc := NewPaperService(paperRepo, tagRepo, nil, t.TempDir())

	err := svc.UpdateTags(context.Background(), userID, paper.ID, []string{"AI", "ML"})
	require.NoError(t, err)

	tags, err := tagRepo.GetByPaperID(context.Background(), paper.ID)
	require.NoError(t, err)
	tagNames := make([]string, len(tags))
	for i, tag := range tags {
		tagNames[i] = tag.Tag
	}
	assert.ElementsMatch(t, []string{"AI", "ML"}, tagNames)
}
```

注意：测试中使用 `NewPaperRepositoryTest` 和 `NewPaperTagRepositoryTest` 来获取具体类型，这需要在 `paper_repository.go` 中导出构造函数（或使用类型断言）。由于现有模式中 repository 已经是通过接口创建的，测试中可以直接使用 `NewPaperRepository(db)` 和 `NewPaperTagRepository(db)` —— 它们返回的就是接口类型，可以被 mock 需要时替换。这里不需要额外的导出函数，直接用 repository 包的构造函数即可。

- [ ] **Step 3: 运行测试确认失败**

Run: `go test ./internal/service/ -run TestPaper -v`
Expected: FAIL — 编译错误

- [ ] **Step 4: 添加错误常量**

在 `internal/service/interfaces.go` 的错误常量区域追加：

```go
// ErrPaperNotFound is returned when a paper is not found
ErrPaperNotFound = errors.New("paper not found")
```

- [ ] **Step 5: 实现 PaperService**

```go
// internal/service/paper_service.go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	pbgrpc "oreader/internal/infra/grpc"
	"oreader/internal/infra/logger"
	"oreader/internal/model"
)

type paperService struct {
	paperRepo PaperRepository
	tagRepo   PaperTagRepository
	grpcClient pbgrpc.PaperConverterClient
	uploadDir string
}

// NewPaperService creates a new paper service
func NewPaperService(
	paperRepo PaperRepository,
	tagRepo PaperTagRepository,
	grpcClient pbgrpc.PaperConverterClient,
	uploadDir string,
) PaperService {
	return &paperService{
		paperRepo:  paperRepo,
		tagRepo:    tagRepo,
		grpcClient: grpcClient,
		uploadDir:  uploadDir,
	}
}

func (s *paperService) UploadPaper(ctx context.Context, userID string, filename string, pdfContent []byte) (*model.Paper, error) {
	logger.Info().
		Str("user_id", userID).
		Str("filename", filename).
		Int("size", len(pdfContent)).
		Msg("Uploading paper")

	// Create upload directory
	dateDir := time.Now().Format("2006-01-02")
	userDir := filepath.Join(s.uploadDir, userID, dateDir)
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Save PDF to disk
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".pdf"
	}
	safeName := strings.ReplaceAll(strings.TrimSuffix(filename, ext), " ", "_")
	pdfPath := filepath.Join(userDir, safeName+ext)
	// Handle duplicate filenames
	counter := 1
	for {
		if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
			break
		}
		pdfPath = filepath.Join(userDir, fmt.Sprintf("%s_%d%s", safeName, counter, ext))
		counter++
	}

	if err := os.WriteFile(pdfPath, pdfContent, 0644); err != nil {
		return nil, fmt.Errorf("failed to save PDF: %w", err)
	}

	// Create Paper record
	paper := &model.Paper{
		UserID:           userID,
		OriginalFilename: filename,
		PDFPath:          pdfPath,
		PDFSize:          int64(len(pdfContent)),
		Status:           model.PaperStatusPending,
	}
	if err := paper.GenerateID(); err != nil {
		return nil, err
	}
	if err := s.paperRepo.Create(ctx, paper); err != nil {
		os.Remove(pdfPath)
		return nil, fmt.Errorf("failed to create paper record: %w", err)
	}

	// Start async conversion
	go s.processConversion(context.Background(), paper.ID, pdfContent, filename)

	logger.Info().
		Str("paper_id", paper.ID).
		Str("pdf_path", pdfPath).
		Msg("Paper uploaded, conversion started")

	return paper, nil
}

func (s *paperService) processConversion(ctx context.Context, paperID string, pdfContent []byte, filename string) {
	// Update status to processing
	paper, err := s.paperRepo.GetByID(ctx, paperID)
	if err != nil || paper == nil {
		logger.Error().Str("paper_id", paperID).Msg("Paper not found for conversion")
		return
	}
	paper.Status = model.PaperStatusProcessing
	s.paperRepo.Update(ctx, paper)

	if s.grpcClient == nil {
		s.failPaper(ctx, paper, "converter service not available")
		return
	}

	// Call gRPC Convert (streaming)
	progress, err := s.grpcClient.Convert(ctx, pdfContent, filename)
	if err != nil {
		s.failPaper(ctx, paper, fmt.Sprintf("conversion failed: %v", err))
		return
	}

	// Extract final markdown
	var markdown string
	for _, p := range progress {
		if p.Error != "" {
			s.failPaper(ctx, paper, p.Error)
			return
		}
		if p.Markdown != "" {
			markdown = p.Markdown
		}
	}

	if markdown == "" {
		s.failPaper(ctx, paper, "no markdown content produced")
		return
	}

	// Extract metadata
	metadata, err := s.grpcClient.ExtractMetadata(ctx, markdown)
	if err != nil {
		logger.Warn().Err(err).Msg("Metadata extraction failed, using raw conversion")
		// Save markdown without metadata
		paper.MarkdownContent = markdown
		paper.Status = model.PaperStatusCompleted
		s.paperRepo.Update(ctx, paper)
		return
	}

	// Update paper with metadata
	authorsJSON, _ := json.Marshal(metadata.Authors)
	keywordsJSON, _ := json.Marshal(metadata.Keywords)

	paper.Title = metadata.Title
	paper.Authors = string(authorsJSON)
	paper.Abstract = metadata.Abstract
	paper.Keywords = string(keywordsJSON)
	paper.PublishedYear = metadata.PublishedYear
	paper.DOI = metadata.DOI
	paper.MarkdownContent = markdown
	paper.Status = model.PaperStatusCompleted

	if err := s.paperRepo.Update(ctx, paper); err != nil {
		logger.Error().Err(err).Str("paper_id", paperID).Msg("Failed to update paper after conversion")
	}

	// Auto-create tags from keywords
	if len(metadata.Keywords) > 0 && s.tagRepo != nil {
		s.tagRepo.SetTags(ctx, paperID, metadata.Keywords)
	}

	logger.Info().
		Str("paper_id", paperID).
		Str("title", metadata.Title).
		Msg("Paper conversion completed")
}

func (s *paperService) failPaper(ctx context.Context, paper *model.Paper, errMsg string) {
	paper.Status = model.PaperStatusFailed
	paper.Error = errMsg
	if err := s.paperRepo.Update(ctx, paper); err != nil {
		logger.Error().Err(err).Str("paper_id", paper.ID).Msg("Failed to update paper status to failed")
	}
	logger.Error().Str("paper_id", paper.ID).Str("error", errMsg).Msg("Paper conversion failed")
}

func (s *paperService) GetPaper(ctx context.Context, userID, paperID string) (*model.Paper, error) {
	paper, err := s.paperRepo.GetByID(ctx, paperID)
	if err != nil {
		return nil, err
	}
	if paper == nil || paper.UserID != userID {
		return nil, ErrPaperNotFound
	}
	return paper, nil
}

func (s *paperService) ListPapers(ctx context.Context, userID string, opts PaperListOptions) ([]*model.Paper, int64, error) {
	papers, total, err := s.paperRepo.ListByUserID(ctx, userID, opts)
	return papers, total, err
}

func (s *paperService) UpdatePaper(ctx context.Context, userID, paperID string, updates map[string]interface{}) (*model.Paper, error) {
	paper, err := s.GetPaper(ctx, userID, paperID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if title, ok := updates["title"]; ok {
		paper.Title = title.(string)
	}
	if abstract, ok := updates["abstract"]; ok {
		paper.Abstract = abstract.(string)
	}
	if publishedYear, ok := updates["published_year"]; ok {
		paper.PublishedYear = publishedYear.(string)
	}
	if doi, ok := updates["doi"]; ok {
		paper.DOI = doi.(string)
	}

	if err := s.paperRepo.Update(ctx, paper); err != nil {
		return nil, err
	}
	return paper, nil
}

func (s *paperService) DeletePaper(ctx context.Context, userID, paperID string) error {
	paper, err := s.GetPaper(ctx, userID, paperID)
	if err != nil {
		return err
	}

	// Delete PDF file from disk
	if paper.PDFPath != "" {
		os.Remove(paper.PDFPath)
	}

	// Delete from database (cascades to tags)
	return s.paperRepo.Delete(ctx, paperID)
}

func (s *paperService) GetPaperStatus(ctx context.Context, userID, paperID string) (*PaperStatusResponse, error) {
	paper, err := s.GetPaper(ctx, userID, paperID)
	if err != nil {
		return nil, err
	}

	progress := int32(0)
	switch paper.Status {
	case model.PaperStatusPending:
		progress = 0
	case model.PaperStatusProcessing:
		progress = 50 // Approximate
	case model.PaperStatusCompleted:
		progress = 100
	}

	return &PaperStatusResponse{
		ID:        paper.ID,
		Status:    paper.Status,
		Progress:  progress,
		Error:     paper.Error,
		CreatedAt: paper.CreatedAt.Format(time.RFC3339),
		UpdatedAt: paper.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *paperService) RetryPaper(ctx context.Context, userID, paperID string) (*model.Paper, error) {
	paper, err := s.GetPaper(ctx, userID, paperID)
	if err != nil {
		return nil, err
	}

	if paper.Status != model.PaperStatusFailed {
		return nil, fmt.Errorf("only failed papers can be retried")
	}

	// Read PDF from disk
	pdfContent, err := os.ReadFile(paper.PDFPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF file: %w", err)
	}

	// Reset status
	paper.Status = model.PaperStatusPending
	paper.Error = ""
	s.paperRepo.Update(ctx, paper)

	// Restart conversion
	go s.processConversion(context.Background(), paperID, pdfContent, paper.OriginalFilename)

	return paper, nil
}

func (s *paperService) ListTags(ctx context.Context, userID string) ([]string, error) {
	return s.paperRepo.ListTags(ctx, userID)
}

func (s *paperService) UpdateTags(ctx context.Context, userID, paperID string, tags []string) error {
	if _, err := s.GetPaper(ctx, userID, paperID); err != nil {
		return err
	}
	if s.tagRepo == nil {
		return fmt.Errorf("tag repository not available")
	}
	return s.tagRepo.SetTags(ctx, paperID, tags)
}

func (s *paperService) DownloadPaper(ctx context.Context, userID, paperID string) (string, string, error) {
	paper, err := s.GetPaper(ctx, userID, paperID)
	if err != nil {
		return "", "", err
	}
	if paper.PDFPath == "" {
		return "", "", fmt.Errorf("PDF file not found")
	}
	return paper.PDFPath, paper.OriginalFilename, nil
}
```

注意：测试代码中引用了 `NewPaperRepositoryTest` / `NewPaperTagRepositoryTest`，实际测试中直接使用 `repository.NewPaperRepository(db)` / `repository.NewPaperTagRepository(db)` 即可，因为它们返回的就是接口类型。需要 import repository 包。

- [ ] **Step 6: 运行测试确认通过**

Run: `go test ./internal/service/ -run TestPaper -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/service/paper_service.go internal/service/paper_service_test.go internal/service/interfaces.go
git commit -m "feat(papers): add PaperService with upload, convert, CRUD, and tag management"
```

---

## Task 7: Paper Handler

**Files:**
- Create: `internal/handler/paper_handler.go`
- Test: `internal/handler/paper_handler_test.go`

- [ ] **Step 1: 写失败测试**

```go
// internal/handler/paper_handler_test.go
package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"oreader/internal/model"
	"oreader/internal/service"
)

// mockPaperService for handler testing
type mockPaperService struct {
	mock.Mock
}

func (m *mockPaperService) UploadPaper(ctx context.Context, userID string, filename string, pdfContent []byte) (*model.Paper, error) {
	args := m.Called(ctx, userID, filename, pdfContent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Paper), args.Error(1)
}
func (m *mockPaperService) GetPaper(ctx context.Context, userID, paperID string) (*model.Paper, error) {
	args := m.Called(ctx, userID, paperID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Paper), args.Error(1)
}
func (m *mockPaperService) ListPapers(ctx context.Context, userID string, opts service.PaperListOptions) ([]*model.Paper, int64, error) {
	args := m.Called(ctx, userID, opts)
	return args.Get(0).([]*model.Paper), args.Get(1).(int64), args.Error(2)
}
func (m *mockPaperService) UpdatePaper(ctx context.Context, userID, paperID string, updates map[string]interface{}) (*model.Paper, error) {
	args := m.Called(ctx, userID, paperID, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Paper), args.Error(1)
}
func (m *mockPaperService) DeletePaper(ctx context.Context, userID, paperID string) error {
	args := m.Called(ctx, userID, paperID)
	return args.Error(0)
}
func (m *mockPaperService) GetPaperStatus(ctx context.Context, userID, paperID string) (*service.PaperStatusResponse, error) {
	args := m.Called(ctx, userID, paperID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.PaperStatusResponse), args.Error(1)
}
func (m *mockPaperService) RetryPaper(ctx context.Context, userID, paperID string) (*model.Paper, error) {
	args := m.Called(ctx, userID, paperID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Paper), args.Error(1)
}
func (m *mockPaperService) ListTags(ctx context.Context, userID string) ([]string, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]string), args.Error(1)
}
func (m *mockPaperService) UpdateTags(ctx context.Context, userID, paperID string, tags []string) error {
	args := m.Called(ctx, userID, paperID, tags)
	return args.Error(0)
}
func (m *mockPaperService) DownloadPaper(ctx context.Context, userID, paperID string) (string, string, error) {
	args := m.Called(ctx, userID, paperID)
	return args.String(0), args.String(1), args.Error(2)
}

func setupPaperRouter(svc service.PaperService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user-id")
		c.Next()
	})

	h := NewPaperHandler(svc)
	papers := router.Group("/api/v1/papers")
	{
		papers.POST("/upload", h.UploadPaper)
		papers.GET("", h.ListPapers)
		papers.GET("/tags", h.ListTags)
		papers.GET("/:id", h.GetPaper)
		papers.GET("/:id/status", h.GetPaperStatus)
		papers.PUT("/:id", h.UpdatePaper)
		papers.PUT("/:id/tags", h.UpdateTags)
		papers.POST("/:id/retry", h.RetryPaper)
		papers.DELETE("/:id", h.DeletePaper)
		papers.GET("/:id/download", h.DownloadPaper)
	}

	return router
}

func TestPaperHandler_UploadPaper(t *testing.T) {
	svc := new(mockPaperService)
	paper := &model.Paper{ID: "paper-1", Status: model.PaperStatusPending, OriginalFilename: "test.pdf"}
	svc.On("UploadPaper", mock.Anything, "test-user-id", "test.pdf", mock.Anything).Return(paper, nil)

	router := setupPaperRouter(svc)

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "test.pdf")
	require.NoError(t, err)
	_, err = part.Write([]byte("fake-pdf-content"))
	require.NoError(t, err)
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/papers/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContent())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "paper-1", resp["id"])
	assert.Equal(t, "pending", resp["status"])
}

func TestPaperHandler_UploadPaper_NoFile(t *testing.T) {
	svc := new(mockPaperService)
	router := setupPaperRouter(svc)

	req := httptest.NewRequest("POST", "/api/v1/papers/upload", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPaperHandler_ListPapers(t *testing.T) {
	svc := new(mockPaperService)
	svc.On("ListPapers", mock.Anything, "test-user-id", mock.Anything).
		Return([]*model.Paper{{Base: model.Base{ID: "p1"}, Title: "Paper 1"}}, int64(1), nil)

	router := setupPaperRouter(svc)

	req := httptest.NewRequest("GET", "/api/v1/papers?limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(1), resp["total"])
}

func TestPaperHandler_GetPaper(t *testing.T) {
	svc := new(mockPaperService)
	svc.On("GetPaper", mock.Anything, "test-user-id", "paper-1").
		Return(&model.Paper{Base: model.Base{ID: "paper-1"}, Title: "My Paper"}, nil)

	router := setupPaperRouter(svc)

	req := httptest.NewRequest("GET", "/api/v1/papers/paper-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPaperHandler_DeletePaper(t *testing.T) {
	svc := new(mockPaperService)
	svc.On("DeletePaper", mock.Anything, "test-user-id", "paper-1").Return(nil)

	router := setupPaperRouter(svc)

	req := httptest.NewRequest("DELETE", "/api/v1/papers/paper-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestPaperHandler_DownloadPaper(t *testing.T) {
	tmpDir := t.TempDir()
	pdfPath := filepath.Join(tmpDir, "test.pdf")
	require.NoError(t, os.WriteFile(pdfPath, []byte("fake-pdf"), 0644))

	svc := new(mockPaperService)
	svc.On("DownloadPaper", mock.Anything, "test-user-id", "paper-1").
		Return(pdfPath, "test.pdf", nil)

	router := setupPaperRouter(svc)

	req := httptest.NewRequest("GET", "/api/v1/papers/paper-1/download", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/handler/ -run TestPaper -v`
Expected: FAIL — 编译错误

- [ ] **Step 3: 实现 PaperHandler**

```go
// internal/handler/paper_handler.go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	apperrors "oreader/internal/infra/errors"
	"oreader/internal/service"
)

// PaperHandler handles paper HTTP requests
type PaperHandler struct {
	paperService service.PaperService
}

// NewPaperHandler creates a new paper handler
func NewPaperHandler(paperService service.PaperService) *PaperHandler {
	return &PaperHandler{paperService: paperService}
}

// UploadPaper handles POST /api/v1/papers/upload
func (h *PaperHandler) UploadPaper(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "No file uploaded", nil)
		return
	}

	src, err := file.Open()
	if err != nil {
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to read file", nil)
		return
	}
	defer src.Close()

	// Read file content (limit to prevent OOM)
	buf := make([]byte, file.Size)
	n, err := src.Read(buf)
	if err != nil {
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to read file content", nil)
		return
	}

	paper, err := h.paperService.UploadPaper(c.Request.Context(), userID.(string), file.Filename, buf[:n])
	if err != nil {
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to upload paper", nil)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"id":                paper.ID,
		"status":            paper.Status,
		"original_filename": paper.OriginalFilename,
	})
}

// ListPapers handles GET /api/v1/papers
func (h *PaperHandler) ListPapers(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	opts := service.PaperListOptions{
		Query:  c.Query("q"),
		Year:   c.Query("year"),
		Tag:    c.Query("tag"),
		Status: c.Query("status"),
		Sort:   c.DefaultQuery("sort", "created_at"),
		Order:  c.DefaultQuery("order", "desc"),
	}
	if v := c.Query("page"); v != "" {
		if page, err := strconv.Atoi(v); err == nil && page > 0 {
			opts.Limit = 20
			opts.Offset = (page - 1) * 20
		}
	}
	if perPage := c.Query("per_page"); perPage != "" {
		if n, err := strconv.Atoi(perPage); err == nil && n > 0 && n <= 100 {
			opts.Limit = n
		}
	}

	papers, total, err := h.paperService.ListPapers(c.Request.Context(), userID.(string), opts)
	if err != nil {
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to list papers", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"papers": papers,
		"total":  total,
	})
}

// GetPaper handles GET /api/v1/papers/:id
func (h *PaperHandler) GetPaper(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	paperID := c.Param("id")
	paper, err := h.paperService.GetPaper(c.Request.Context(), userID.(string), paperID)
	if err != nil {
		apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Paper not found", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"paper": paper})
}

// GetPaperStatus handles GET /api/v1/papers/:id/status
func (h *PaperHandler) GetPaperStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	status, err := h.paperService.GetPaperStatus(c.Request.Context(), userID.(string), c.Param("id"))
	if err != nil {
		apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Paper not found", nil)
		return
	}

	c.JSON(http.StatusOK, status)
}

// UpdatePaper handles PUT /api/v1/papers/:id
func (h *PaperHandler) UpdatePaper(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	var req struct {
		Title         *string `json:"title"`
		Abstract      *string `json:"abstract"`
		PublishedYear *string `json:"published_year"`
		DOI           *string `json:"doi"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "Invalid request body", nil)
		return
	}

	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Abstract != nil {
		updates["abstract"] = *req.Abstract
	}
	if req.PublishedYear != nil {
		updates["published_year"] = *req.PublishedYear
	}
	if req.DOI != nil {
		updates["doi"] = *req.DOI
	}

	paper, err := h.paperService.UpdatePaper(c.Request.Context(), userID.(string), c.Param("id"), updates)
	if err != nil {
		apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Paper not found", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"paper": paper})
}

// UpdateTags handles PUT /api/v1/papers/:id/tags
func (h *PaperHandler) UpdateTags(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	var req struct {
		Tags []string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "Invalid request body", nil)
		return
	}

	if err := h.paperService.UpdateTags(c.Request.Context(), userID.(string), c.Param("id"), req.Tags); err != nil {
		apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tags updated"})
}

// RetryPaper handles POST /api/v1/papers/:id/retry
func (h *PaperHandler) RetryPaper(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	paper, err := h.paperService.RetryPaper(c.Request.Context(), userID.(string), c.Param("id"))
	if err != nil {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, err.Error(), nil)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"paper": paper})
}

// DeletePaper handles DELETE /api/v1/papers/:id
func (h *PaperHandler) DeletePaper(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	if err := h.paperService.DeletePaper(c.Request.Context(), userID.(string), c.Param("id")); err != nil {
		apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, err.Error(), nil)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListTags handles GET /api/v1/papers/tags
func (h *PaperHandler) ListTags(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	tags, err := h.paperService.ListTags(c.Request.Context(), userID.(string))
	if err != nil {
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to list tags", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

// DownloadPaper handles GET /api/v1/papers/:id/download
func (h *PaperHandler) DownloadPaper(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	pdfPath, filename, err := h.paperService.DownloadPaper(c.Request.Context(), userID.(string), c.Param("id"))
	if err != nil {
		apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, err.Error(), nil)
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.File(pdfPath)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/handler/ -run TestPaper -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/handler/paper_handler.go internal/handler/paper_handler_test.go
git commit -m "feat(papers): add PaperHandler with all API endpoints"
```

---

## Task 8: 注册路由 + AutoMigrate

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: 在 main.go 中注册 Paper 相关依赖**

在 import 区域添加：
```go
pbgrpc "oreader/internal/infra/grpc"
```

在 AutoMigrate 中追加：
```go
&model.Paper{},
&model.PaperTag{},
&model.PaperCollection{},
&model.PaperCollectionItem{},
```

在 repository 初始化区域追加：
```go
paperRepo := repository.NewPaperRepository(db)
paperTagRepo := repository.NewPaperTagRepository(db)
```

在 service 初始化区域追加：
```go
// Initialize paper converter gRPC client
var paperGRPCClient pbgrpc.PaperConverterClient
grpcTimeout, err := cfg.GetGRPCTimeout()
if err != nil {
	logger.Warn().Err(err).Msg("Invalid gRPC timeout, using default 5m")
	grpcTimeout = 5 * time.Minute
}
paperGRPCClient, err = pbgrpc.NewPaperClient(cfg.Paper.GRPCAddr, grpcTimeout)
if err != nil {
	logger.Warn().Err(err).Msg("Paper converter gRPC service not available, paper upload will be limited")
}

paperService := service.NewPaperService(paperRepo, paperTagRepo, paperGRPCClient, cfg.Paper.UploadDir)
```

在 handler 初始化区域追加：
```go
paperHandler := handler.NewPaperHandler(paperService)
```

在 protected 路由组中追加：
```go
// Paper routes
papers := protected.Group("/papers")
{
    papers.POST("/upload", paperHandler.UploadPaper)
    papers.GET("", paperHandler.ListPapers)
    papers.GET("/tags", paperHandler.ListTags)
    papers.GET("/:id", paperHandler.GetPaper)
    papers.GET("/:id/status", paperHandler.GetPaperStatus)
    papers.PUT("/:id", paperHandler.UpdatePaper)
    papers.PUT("/:id/tags", paperHandler.UpdateTags)
    papers.POST("/:id/retry", paperHandler.RetryPaper)
    papers.DELETE("/:id", paperHandler.DeletePaper)
    papers.GET("/:id/download", paperHandler.DownloadPaper)
}
```

在 graceful shutdown 区域追加 gRPC 连接关闭：
```go
if paperGRPCClient != nil {
    paperGRPCClient.Close()
}
```

- [ ] **Step 2: 编译确认无错误**

Run: `go build ./cmd/server/...`
Expected: 成功

- [ ] **Step 3: 运行全部后端测试**

Run: `go test ./internal/... -v -count=1`
Expected: 全部 PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat(papers): register paper routes and auto-migrate in main.go"
```

---

## Task 9: Python gRPC 服务

**Files:**
- Create: `converter/server.py`
- Create: `converter/converter.py`
- Create: `converter/requirements.txt`
- Create: `converter/tests/__init__.py`
- Create: `converter/tests/test_converter.py`

- [ ] **Step 1: 创建 requirements.txt**

```
grpcio>=1.60.0
grpcio-tools>=1.60.0
magic-pdf[full]>=0.10.0
openai>=1.0.0
python-dotenv>=1.0.0
```

- [ ] **Step 2: 写 converter.py 的测试**

```python
# converter/tests/test_converter.py
import json
import unittest
from unittest.mock import patch, MagicMock

import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

from converter import extract_metadata_prompt, parse_metadata_response


class TestConverter(unittest.TestCase):

    def test_extract_metadata_prompt_contains_required_fields(self):
        prompt = extract_metadata_prompt("# Test Paper\n\nSome content")
        self.assertIn("title", prompt.lower())
        self.assertIn("authors", prompt.lower())
        self.assertIn("abstract", prompt.lower())
        self.assertIn("keywords", prompt.lower())

    def test_parse_metadata_response_valid_json(self):
        response = json.dumps({
            "title": "Test Paper",
            "authors": ["Alice", "Bob"],
            "abstract": "This is abstract",
            "keywords": ["ML", "AI"],
            "published_year": "2024",
            "doi": "10.1234/test"
        })
        result = parse_metadata_response(response)
        self.assertEqual(result.title, "Test Paper")
        self.assertEqual(result.authors, ["Alice", "Bob"])
        self.assertEqual(result.published_year, "2024")

    def test_parse_metadata_response_partial(self):
        response = json.dumps({"title": "Only Title"})
        result = parse_metadata_response(response)
        self.assertEqual(result.title, "Only Title")
        self.assertEqual(result.authors, [])
        self.assertEqual(result.keywords, [])

    def test_parse_metadata_response_invalid_json(self):
        result = parse_metadata_response("not json at all")
        self.assertIsNone(result)


if __name__ == '__main__':
    unittest.main()
```

- [ ] **Step 3: 运行测试确认失败**

Run: `cd converter && python -m pytest tests/ -v`
Expected: FAIL — 模块不存在

- [ ] **Step 4: 实现 converter.py**

```python
# converter/converter.py
import json
import logging
import os
import re
from dataclasses import dataclass, field

import openai

logger = logging.getLogger(__name__)

METADATA_EXTRACTION_PROMPT = """You are an academic paper metadata extractor. Given the following Markdown content converted from a PDF paper, extract structured metadata.

IMPORTANT: Return ONLY a JSON object, no markdown fences or explanation.

Required fields:
- title: Paper title (string)
- authors: List of author names (array of strings)
- abstract: Paper abstract (string, may be empty)
- keywords: List of keywords/topics (array of strings, 3-8 items)
- published_year: Publication year (string, e.g. "2024")
- doi: DOI if found (string, may be empty)

Markdown content:
{markdown}

JSON response:"""


@dataclass
class PaperMetadataResult:
    title: str = ""
    authors: list = field(default_factory=list)
    abstract: str = ""
    keywords: list = field(default_factory=list)
    published_year: str = ""
    doi: str = ""


def extract_metadata_prompt(markdown: str) -> str:
    return METADATA_EXTRACTION_PROMPT.format(markdown=markdown[:15000])


def parse_metadata_response(response: str) -> PaperMetadataResult | None:
    """Parse LLM response into PaperMetadataResult."""
    # Strip markdown code fences if present
    cleaned = response.strip()
    if cleaned.startswith("```"):
        cleaned = re.sub(r'^```\w*\n?', '', cleaned)
        cleaned = re.sub(r'\n?```$', '', cleaned)

    try:
        data = json.loads(cleaned)
    except json.JSONDecodeError:
        logger.error("Failed to parse metadata JSON: %s", cleaned[:200])
        return None

    return PaperMetadataResult(
        title=data.get("title", ""),
        authors=data.get("authors", []),
        abstract=data.get("abstract", ""),
        keywords=data.get("keywords", []),
        published_year=str(data.get("published_year", "")),
        doi=data.get("doi", ""),
    )


def extract_metadata(markdown: str) -> PaperMetadataResult:
    """Call LLM to extract metadata from markdown content."""
    api_key = os.getenv("LLM_API_KEY")
    base_url = os.getenv("LLM_BASE_URL", "https://api.openai.com/v1")
    model = os.getenv("LLM_MODEL", "gpt-4o-mini")

    if not api_key:
        logger.warning("LLM_API_KEY not set, skipping metadata extraction")
        return PaperMetadataResult()

    client = openai.OpenAI(api_key=api_key, base_url=base_url)
    prompt = extract_metadata_prompt(markdown)

    try:
        response = client.chat.completions.create(
            model=model,
            messages=[{"role": "user", "content": prompt}],
            temperature=0.0,
            max_tokens=2000,
        )
        content = response.choices[0].message.content or ""
        return parse_metadata_response(content)
    except Exception as e:
        logger.error("LLM metadata extraction failed: %s", e)
        return PaperMetadataResult()


def refine_markdown(markdown: str) -> str:
    """Call LLM to fix formatting issues in the markdown."""
    api_key = os.getenv("LLM_API_KEY")
    base_url = os.getenv("LLM_BASE_URL", "https://api.openai.com/v1")
    model = os.getenv("LLM_MODEL", "gpt-4o-mini")

    if not api_key:
        return markdown

    client = openai.OpenAI(api_key=api_key, base_url=base_url)

    prompt = f"""Fix formatting issues in this Markdown converted from a PDF academic paper.
Rules:
- Fix broken LaTeX formulas (ensure $...$ and $$...$$ are properly paired)
- Fix table formatting
- Remove page numbers, headers, footers
- Fix paragraph breaks
- Keep all content, don't remove anything important
- Return the corrected markdown only

Markdown:
{markdown[:20000]}"""

    try:
        response = client.chat.completions.create(
            model=model,
            messages=[{"role": "user", "content": prompt}],
            temperature=0.0,
            max_tokens=16000,
        )
        return response.choices[0].message.content or markdown
    except Exception as e:
        logger.error("LLM markdown refinement failed: %s", e)
        return markdown
```

- [ ] **Step 5: 实现 server.py**

```python
# converter/server.py
import logging
import os
import sys
from concurrent import futures

import grpc

# Add proto to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'proto'))
import paper_pb2
import paper_pb2_grpc

from converter import extract_metadata, refine_markdown

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class PaperConverterServicer(paper_pb2_grpc.PaperConverterServicer):

    def Convert(self, request, context):
        """Convert PDF to Markdown with streaming progress."""
        logger.info("Converting PDF: %s (%d bytes)", request.filename, len(request.pdf_content))

        # Phase 1: Mining (MinerU)
        yield paper_pb2.ConvertProgress(status="mining", progress=10)
        try:
            from magic_pdf.data.data_reader_writer import FileBasedDataWriter, FileBasedDataReader
            import tempfile

            # Save PDF to temp file
            with tempfile.NamedTemporaryFile(suffix='.pdf', delete=False) as tmp:
                tmp.write(request.pdf_content)
                tmp_path = tmp.name

            try:
                # Use MinerU to convert
                output_dir = tempfile.mkdtemp()
                reader = FileBasedDataReader("")
                writer = FileBasedDataWriter(output_dir)

                # Call MinerU pipeline
                from magic_pdf.pipe.UNIPipe import UNIPipe
                from magic_pdf.libs.draw_bbox import draw_layout_bbox

                pipe = UNIPipe(tmp_path, "", reader, writer)
                pipe.pipe_classify()
                pipe.pipe_parse()
                pipe.pipe_mk_markdown(output_dir, drop_mode="none")

                # Read generated markdown
                md_path = os.path.join(output_dir, "auto", os.path.splitext(request.filename)[0] + ".md")
                if os.path.exists(md_path):
                    with open(md_path, 'r') as f:
                        markdown = f.read()
                else:
                    # Fallback: try to find any .md file
                    md_files = []
                    for root, dirs, files in os.walk(output_dir):
                        for f in files:
                            if f.endswith('.md'):
                                md_files.append(os.path.join(root, f))
                    if md_files:
                        with open(md_files[0], 'r') as f:
                            markdown = f.read()
                    else:
                        markdown = "# Conversion Error\n\nMinerU did not produce markdown output."
            finally:
                os.unlink(tmp_path)

            yield paper_pb2.ConvertProgress(status="mining", progress=50)

        except Exception as e:
            logger.error("MinerU conversion failed: %s", e)
            yield paper_pb2.ConvertProgress(
                status="failed", progress=0,
                error=f"PDF mining failed: {str(e)}"
            )
            return

        # Phase 2: LLM refinement
        yield paper_pb2.ConvertProgress(status="llm_refining", progress=70)
        markdown = refine_markdown(markdown)
        yield paper_pb2.ConvertProgress(status="llm_refining", progress=90)

        # Phase 3: Complete
        yield paper_pb2.ConvertProgress(
            status="completed", progress=100,
            markdown=markdown
        )
        logger.info("Conversion completed for: %s", request.filename)

    def ExtractMetadata(self, request, context):
        """Extract metadata from markdown content."""
        logger.info("Extracting metadata from markdown (%d chars)", len(request.markdown))

        result = extract_metadata(request.markdown)

        return paper_pb2.PaperMetadata(
            title=result.title,
            authors=result.authors,
            abstract=result.abstract,
            keywords=result.keywords,
            published_year=result.published_year,
            doi=result.doi,
        )


def serve():
    port = os.getenv("GRPC_PORT", "50051")
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))
    paper_pb2_grpc.add_PaperConverterServicer_to_server(
        PaperConverterServicer(), server
    )
    server.add_insecure_port(f'[::]:{port}')
    server.start()
    logger.info("Paper converter gRPC server started on port %s", port)
    server.wait_for_termination()


if __name__ == '__main__':
    serve()
```

- [ ] **Step 6: 创建 tests/__init__.py**

```python
# converter/tests/__init__.py
```

- [ ] **Step 7: 运行测试确认通过**

Run: `cd converter && python -m pytest tests/ -v`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add converter/
git commit -m "feat(papers): add Python gRPC service with MinerU + LLM conversion"
```

---

## Task 10: 前端 — 类型定义 + API Hooks

**Files:**
- Create: `web/src/types/paper.ts`
- Create: `web/src/hooks/usePapers.ts`

- [ ] **Step 1: 创建论文类型定义**

```typescript
// web/src/types/paper.ts
export interface Paper {
  id: string
  user_id: string
  title: string
  authors: string | null       // JSON array string
  abstract: string | null
  keywords: string | null       // JSON array string
  published_year: string | null
  doi: string | null
  pdf_path: string | null
  pdf_size: number
  markdown_content: string | null
  cover_image: string | null
  original_filename: string
  status: PaperStatus
  error: string | null
  created_at: string
  updated_at: string
}

export type PaperStatus = 'pending' | 'processing' | 'completed' | 'failed'

export interface PaperTag {
  id: string
  paper_id: string
  tag: string
}

export interface ListPapersResponse {
  papers: Paper[]
  total: number
}

export interface ListPapersOptions {
  page?: number
  per_page?: number
  q?: string
  year?: string
  tag?: string
  status?: PaperStatus
  sort?: string
  order?: 'asc' | 'desc'
}

export interface PaperStatusResponse {
  id: string
  status: PaperStatus
  progress: number
  error?: string
  created_at: string
  updated_at: string
}

export interface UpdatePaperRequest {
  title?: string
  abstract?: string
  published_year?: string
  doi?: string
}

export interface UpdateTagsRequest {
  tags: string[]
}

export interface UploadPaperResponse {
  id: string
  status: PaperStatus
  original_filename: string
}
```

- [ ] **Step 2: 创建 API Hooks**

```typescript
// web/src/hooks/usePapers.ts
import { useMutation, useQuery } from '@tanstack/react-query'
import apiClient from '@/lib/api/axios'
import type {
  ListPapersOptions,
  ListPapersResponse,
  Paper,
  PaperStatusResponse,
  UpdatePaperRequest,
  UpdateTagsRequest,
  UploadPaperResponse,
} from '@/types/paper'

// API functions
async function uploadPaper(file: File): Promise<UploadPaperResponse> {
  const formData = new FormData()
  formData.append('file', file)
  const response = await apiClient.post<UploadPaperResponse>('/papers/upload', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return response.data
}

async function listPapers(options?: ListPapersOptions): Promise<ListPapersResponse> {
  const params = new URLSearchParams()
  if (options?.page) params.append('page', options.page.toString())
  if (options?.per_page) params.append('per_page', options.per_page.toString())
  if (options?.q) params.append('q', options.q)
  if (options?.year) params.append('year', options.year)
  if (options?.tag) params.append('tag', options.tag)
  if (options?.status) params.append('status', options.status)
  if (options?.sort) params.append('sort', options.sort)
  if (options?.order) params.append('order', options.order)

  const url = params.toString() ? `/papers?${params.toString()}` : '/papers'
  const response = await apiClient.get<ListPapersResponse>(url)
  return response.data
}

async function getPaper(paperId: string): Promise<{ paper: Paper }> {
  const response = await apiClient.get<{ paper: Paper }>(`/papers/${paperId}`)
  return response.data
}

async function getPaperStatus(paperId: string): Promise<PaperStatusResponse> {
  const response = await apiClient.get<PaperStatusResponse>(`/papers/${paperId}/status`)
  return response.data
}

async function updatePaper(paperId: string, data: UpdatePaperRequest): Promise<{ paper: Paper }> {
  const response = await apiClient.put<{ paper: Paper }>(`/papers/${paperId}`, data)
  return response.data
}

async function updateTags(paperId: string, data: UpdateTagsRequest): Promise<void> {
  await apiClient.put(`/papers/${paperId}/tags`, data)
}

async function deletePaper(paperId: string): Promise<void> {
  await apiClient.delete(`/papers/${paperId}`)
}

async function retryPaper(paperId: string): Promise<{ paper: Paper }> {
  const response = await apiClient.post<{ paper: Paper }>(`/papers/${paperId}/retry`)
  return response.data
}

async function listTags(): Promise<{ tags: string[] }> {
  const response = await apiClient.get<{ tags: string[] }>('/papers/tags')
  return response.data
}

export function usePapers() {
  const useUploadPaper = () =>
    useMutation({
      mutationFn: uploadPaper,
    })

  const useListPapers = (options?: ListPapersOptions) =>
    useQuery({
      queryKey: ['papers', 'list', options],
      queryFn: () => listPapers(options),
      staleTime: 2 * 60 * 1000,
    })

  const useGetPaper = (paperId: string | null) =>
    useQuery({
      queryKey: ['papers', paperId],
      queryFn: () => getPaper(paperId!),
      enabled: !!paperId,
      staleTime: 5 * 60 * 1000,
    })

  const useGetPaperStatus = (paperId: string | null) =>
    useQuery({
      queryKey: ['papers', 'status', paperId],
      queryFn: () => getPaperStatus(paperId!),
      enabled: !!paperId,
      refetchInterval: (query) => {
        const data = query.state.data
        if (!data) return false
        return data.status === 'pending' || data.status === 'processing' ? 3000 : false
      },
    })

  const useUpdatePaper = () =>
    useMutation({
      mutationFn: ({ paperId, data }: { paperId: string; data: UpdatePaperRequest }) =>
        updatePaper(paperId, data),
    })

  const useUpdateTags = () =>
    useMutation({
      mutationFn: ({ paperId, tags }: { paperId: string; tags: string[] }) =>
        updateTags(paperId, { tags }),
    })

  const useDeletePaper = () =>
    useMutation({
      mutationFn: deletePaper,
    })

  const useRetryPaper = () =>
    useMutation({
      mutationFn: retryPaper,
    })

  const useListTags = () =>
    useQuery({
      queryKey: ['papers', 'tags'],
      queryFn: listTags,
      staleTime: 5 * 60 * 1000,
    })

  return {
    useUploadPaper,
    useListPapers,
    useGetPaper,
    useGetPaperStatus,
    useUpdatePaper,
    useUpdateTags,
    useDeletePaper,
    useRetryPaper,
    useListTags,
  }
}
```

- [ ] **Step 3: Commit**

```bash
git add web/src/types/paper.ts web/src/hooks/usePapers.ts
git commit -m "feat(papers): add frontend types and TanStack Query hooks for papers API"
```

---

## Task 11: 前端 — 论文列表页

**Files:**
- Create: `web/src/components/papers/PaperList.tsx`
- Create: `web/src/components/papers/PaperUpload.tsx`
- Create: `web/src/pages/papers/PapersPage.tsx`

- [ ] **Step 1: 创建 PaperList 组件**

```tsx
// web/src/components/papers/PaperList.tsx
import { FileText, Loader2, AlertCircle, RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import type { Paper, PaperStatus } from '@/types/paper'

interface PaperListProps {
  papers: Paper[]
  isLoading?: boolean
  onPaperClick: (paperId: string) => void
  onRetry: (paperId: string) => void
  onDelete: (paperId: string) => void
}

function StatusBadge({ status }: { status: PaperStatus }) {
  switch (status) {
    case 'completed':
      return <Badge variant="secondary">Completed</Badge>
    case 'processing':
      return <Badge className="bg-blue-500 text-white">Processing</Badge>
    case 'pending':
      return <Badge className="bg-yellow-500 text-white">Pending</Badge>
    case 'failed':
      return <Badge variant="destructive">Failed</Badge>
    default:
      return <Badge variant="outline">{status}</Badge>
  }
}

function parseJsonArray(str: string | null): string[] {
  if (!str) return []
  try {
    return JSON.parse(str)
  } catch {
    return []
  }
}

export function PaperList({ papers, isLoading, onPaperClick, onRetry, onDelete }: PaperListProps) {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (papers.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center text-muted-foreground">
        <FileText className="h-12 w-12 mb-4" />
        <p>No papers yet</p>
        <p className="text-sm mt-1">Upload a PDF to get started</p>
      </div>
    )
  }

  return (
    <div className="space-y-2">
      {papers.map((paper) => (
        <div
          key={paper.id}
          className="p-4 rounded-lg border hover:bg-accent/50 cursor-pointer transition-colors"
          onClick={() => onPaperClick(paper.id)}
        >
          <div className="flex items-start justify-between gap-3">
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-1">
                <FileText className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                <h3 className="font-medium truncate">{paper.title || paper.original_filename}</h3>
              </div>
              <div className="text-sm text-muted-foreground">
                {parseJsonArray(paper.authors).join(', ') && (
                  <span className="mr-2">{parseJsonArray(paper.authors).slice(0, 3).join(', ')}</span>
                )}
                {paper.published_year && <span>{paper.published_year}</span>}
              </div>
              {paper.abstract && (
                <p className="text-sm text-muted-foreground mt-1 line-clamp-2">{paper.abstract}</p>
              )}
              {parseJsonArray(paper.keywords).length > 0 && (
                <div className="flex gap-1 mt-2 flex-wrap">
                  {parseJsonArray(paper.keywords).slice(0, 3).map((kw) => (
                    <Badge key={kw} variant="outline" className="text-xs">{kw}</Badge>
                  ))}
                </div>
              )}
            </div>
            <div className="flex flex-col items-end gap-2 flex-shrink-0">
              <StatusBadge status={paper.status} />
              {paper.status === 'failed' && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={(e) => { e.stopPropagation(); onRetry(paper.id) }}
                >
                  <RefreshCw className="h-3 w-3" />
                </Button>
              )}
              {paper.status === 'failed' && (
                <p className="text-xs text-destructive max-w-48 truncate">{paper.error}</p>
              )}
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}
```

- [ ] **Step 2: 创建 PaperUpload 模态框组件**

```tsx
// web/src/components/papers/PaperUpload.tsx
import { useState, useCallback, useRef } from 'react'
import { Upload, X, FileText, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'

interface PaperUploadProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess: () => void
}

export function PaperUpload({ open, onOpenChange, onSuccess }: PaperUploadProps) {
  const [file, setFile] = useState<File | null>(null)
  const [dragActive, setDragActive] = useState(false)
  const [isUploading, setIsUploading] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const handleDrag = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true)
    } else if (e.type === 'dragleave') {
      setDragActive(false)
    }
  }, [])

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setDragActive(false)

    const droppedFile = e.dataTransfer.files?.[0]
    if (droppedFile && droppedFile.type === 'application/pdf') {
      setFile(droppedFile)
    }
  }, [])

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const selected = e.target.files?.[0]
    if (selected) {
      setFile(selected)
    }
  }

  const handleUpload = async () => {
    if (!file || isUploading) return

    setIsUploading(true)
    try {
      const formData = new FormData()
      formData.append('file', file)
      const token = document.querySelector('meta[name="csrf-token"]')?.getAttribute('content')
      const response = await fetch('/api/v1/papers/upload', {
        method: 'POST',
        body: formData,
        credentials: 'include',
        headers: token ? { 'X-CSRF-Token': token } : {},
      })

      if (!response.ok) {
        throw new Error('Upload failed')
      }

      setFile(null)
      onOpenChange(false)
      onSuccess()
    } catch (error) {
      console.error('Upload error:', error)
    } finally {
      setIsUploading(false)
    }
  }

  const handleClose = () => {
    if (!isUploading) {
      setFile(null)
      onOpenChange(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Upload Paper</DialogTitle>
        </DialogHeader>

        <div
          className={`border-2 border-dashed rounded-lg p-8 text-center transition-colors ${
            dragActive ? 'border-primary bg-primary/5' : 'border-muted-foreground/25'
          }`}
          onDragEnter={handleDrag}
          onDragLeave={handleDrag}
          onDragOver={handleDrag}
          onDrop={handleDrop}
          onClick={() => inputRef.current?.click()}
        >
          <input
            ref={inputRef}
            type="file"
            accept=".pdf"
            className="hidden"
            onChange={handleFileChange}
          />
          {file ? (
            <div className="flex flex-col items-center gap-2">
              <FileText className="h-10 w-10 text-muted-foreground" />
              <p className="text-sm font-medium">{file.name}</p>
              <p className="text-xs text-muted-foreground">{(file.size / 1024 / 1024).toFixed(2)} MB</p>
              <Button variant="ghost" size="sm" onClick={(e) => { e.stopPropagation(); setFile(null) }}>
                <X className="h-3 w-3 mr-1" /> Remove
              </Button>
            </div>
          ) : (
            <div className="flex flex-col items-center gap-2">
              <Upload className="h-10 w-10 text-muted-foreground" />
              <p className="text-sm text-muted-foreground">
                Drag and drop a PDF here, or click to browse
              </p>
              <p className="text-xs text-muted-foreground">Supports academic papers in PDF format</p>
            </div>
          )}
        </div>

        <Button
          className="w-full"
          disabled={!file || isUploading}
          onClick={handleUpload}
        >
          {isUploading ? (
            <>
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              Uploading...
            </>
          ) : (
            <>
              <Upload className="h-4 w-4 mr-2" />
              Upload
            </>
          )}
        </Button>
      </DialogContent>
    </Dialog>
  )
}
```

- [ ] **Step 3: 创建 PapersPage**

```tsx
// web/src/pages/papers/PapersPage.tsx
import { useState, useCallback } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import { Upload, Search, SlidersHorizontal, ArrowLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { PaperList } from '@/components/papers/PaperList'
import { PaperUpload } from '@/components/papers/PaperUpload'
import { usePapers } from '@/hooks/usePapers'
import { useToast } from '@/components/ui/toast'
import type { ListPapersOptions, PaperStatus } from '@/types/paper'

export function PapersPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const navigate = useNavigate()
  const toast = useToast()
  const [isUploadOpen, setIsUploadOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState(searchParams.get('q') || '')
  const [selectedYear, setSelectedYear] = useState(searchParams.get('year') || '')
  const [selectedTag, setSelectedTag] = useState(searchParams.get('tag') || '')

  const { useListPapers, useDeletePaper, useRetryPaper, useListTags } = usePapers()

  const listOptions: ListPapersOptions = {
    page: parseInt(searchParams.get('page') || '1'),
    per_page: 20,
    q: searchQuery || undefined,
    year: selectedYear || undefined,
    tag: selectedTag || undefined,
    sort: 'created_at',
    order: 'desc',
  }

  const { data: papersData, isLoading, refetch } = useListPapers(listOptions)
  const { data: tagsData } = useListTags()
  const deletePaper = useDeletePaper()
  const retryPaper = useRetryPaper()

  const papers = papersData?.papers ?? []
  const total = papersData?.total ?? 0

  const handleSearch = useCallback((query: string) => {
    setSearchQuery(query)
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      if (query) next.set('q', query)
      else next.delete('q')
      next.set('page', '1')
      return next
    })
  }, [setSearchParams])

  const handlePaperClick = useCallback((paperId: string) => {
    navigate(`/papers/${paperId}`)
  }, [navigate])

  const handleRetry = useCallback((paperId: string) => {
    retryPaper.mutate(paperId, {
      onSuccess: () => {
        toast.showSuccess('Retry started')
        refetch()
      },
      onError: () => {
        toast.showError('Retry failed')
      },
    })
  }, [retryPaper, refetch, toast])

  const handleDelete = useCallback((paperId: string) => {
    if (!confirm('Delete this paper?')) return
    deletePaper.mutate(paperId, {
      onSuccess: () => {
        toast.showSuccess('Paper deleted')
        refetch()
      },
      onError: () => {
        toast.showError('Failed to delete paper')
      },
    })
  }, [deletePaper, refetch, toast])

  const handlePageChange = (page: number) => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      next.set('page', page.toString())
      return next
    })
  }

  const currentPage = parseInt(searchParams.get('page') || '1')
  const totalPages = Math.ceil(total / 20)

  return (
    <div className="max-w-5xl mx-auto py-6 px-4">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <Button variant="ghost" onClick={() => navigate('/items')}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back
          </Button>
          <h1 className="text-2xl font-bold">Papers</h1>
          <span className="text-sm text-muted-foreground">({total})</span>
        </div>
        <Button onClick={() => setIsUploadOpen(true)}>
          <Upload className="h-4 w-4 mr-2" />
          Upload PDF
        </Button>
      </div>

      {/* Search & Filters */}
      <div className="flex gap-3 mb-6">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search papers..."
            value={searchQuery}
            onChange={(e) => handleSearch(e.target.value)}
            className="pl-10"
          />
        </div>
        {tagsData?.tags && tagsData.tags.length > 0 && (
          <div className="flex gap-2 overflow-x-auto">
            {tagsData.tags.slice(0, 5).map((tag) => (
              <Button
                key={tag}
                variant={selectedTag === tag ? 'secondary' : 'outline'}
                size="sm"
                onClick={() => {
                  const newTag = selectedTag === tag ? '' : tag
                  setSelectedTag(newTag)
                  setSearchParams((prev) => {
                    const next = new URLSearchParams(prev)
                    if (newTag) next.set('tag', newTag)
                    else next.delete('tag')
                    next.set('page', '1')
                    return next
                  })
                }}
              >
                {tag}
              </Button>
            ))}
          </div>
        )}
      </div>

      {/* Paper List */}
      <PaperList
        papers={papers}
        isLoading={isLoading}
        onPaperClick={handlePaperClick}
        onRetry={handleRetry}
        onDelete={handleDelete}
      />

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2 mt-6">
          <Button
            variant="outline"
            size="sm"
            disabled={currentPage <= 1}
            onClick={() => handlePageChange(currentPage - 1)}
          >
            Previous
          </Button>
          <span className="text-sm text-muted-foreground">
            Page {currentPage} of {totalPages}
          </span>
          <Button
            variant="outline"
            size="sm"
            disabled={currentPage >= totalPages}
            onClick={() => handlePageChange(currentPage + 1)}
          >
            Next
          </Button>
        </div>
      )}

      {/* Upload Modal */}
      <PaperUpload
        open={isUploadOpen}
        onOpenChange={setIsUploadOpen}
        onSuccess={() => refetch()}
      />
    </div>
  )
}

export default PapersPage
```

- [ ] **Step 4: 前端编译检查**

Run: `cd web && npx tsc --noEmit`
Expected: 无错误

- [ ] **Step 5: Commit**

```bash
git add web/src/components/papers/ web/src/pages/papers/PapersPage.tsx
git commit -m "feat(papers): add PapersPage, PaperList, and PaperUpload components"
```

---

## Task 12: 前端 — 论文详情页

**Files:**
- Create: `web/src/components/papers/PaperMeta.tsx`
- Create: `web/src/pages/papers/PaperViewPage.tsx`

- [ ] **Step 1: 创建 PaperMeta 组件**

```tsx
// web/src/components/papers/PaperMeta.tsx
import { Calendar, Users, ExternalLink, Download, Tag } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import type { Paper } from '@/types/paper'

function parseJsonArray(str: string | null): string[] {
  if (!str) return []
  try { return JSON.parse(str) } catch { return [] }
}

interface PaperMetaProps {
  paper: Paper
  onDownload: () => void
}

export function PaperMeta({ paper, onDownload }: PaperMetaProps) {
  const authors = parseJsonArray(paper.authors)
  const keywords = parseJsonArray(paper.keywords)

  return (
    <div className="space-y-4">
      <h1 className="text-2xl md:text-3xl font-bold">{paper.title || 'Untitled'}</h1>

      <div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
        {authors.length > 0 && (
          <div className="flex items-center gap-1">
            <Users className="h-4 w-4" />
            <span>{authors.join(', ')}</span>
          </div>
        )}
        {paper.published_year && (
          <div className="flex items-center gap-1">
            <Calendar className="h-4 w-4" />
            <span>{paper.published_year}</span>
          </div>
        )}
        {paper.doi && (
          <div className="flex items-center gap-1">
            <ExternalLink className="h-4 w-4" />
            <a
              href={`https://doi.org/${paper.doi}`}
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary hover:underline"
            >
              DOI: {paper.doi}
            </a>
          </div>
        )}
      </div>

      {paper.abstract && (
        <div className="border-l-4 border-muted pl-4">
          <h3 className="font-medium mb-1">Abstract</h3>
          <p className="text-sm text-muted-foreground">{paper.abstract}</p>
        </div>
      )}

      {keywords.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {keywords.map((kw) => (
            <Badge key={kw} variant="secondary" className="flex items-center gap-1">
              <Tag className="h-3 w-3" />
              {kw}
            </Badge>
          ))}
        </div>
      )}

      <div className="flex items-center gap-2">
        <Button variant="outline" size="sm" onClick={onDownload}>
          <Download className="h-4 w-4 mr-2" />
          Download PDF
        </Button>
        <span className="text-xs text-muted-foreground">
          {(paper.pdf_size / 1024 / 1024).toFixed(2)} MB
        </span>
      </div>
    </div>
  )
}
```

- [ ] **Step 2: 创建 PaperViewPage**

```tsx
// web/src/pages/papers/PaperViewPage.tsx
import { useParams, useNavigate } from 'react-router-dom'
import { ArrowLeft, Loader2, AlertCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { MarkdownRenderer } from '@/components/ui/MarkdownRenderer'
import { PaperMeta } from '@/components/papers/PaperMeta'
import { usePapers } from '@/hooks/usePapers'
import { useState } from 'react'

export default function PaperViewPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { useGetPaper, useGetPaperStatus } = usePapers()
  const [isEditing, setIsEditing] = useState(false)

  const { data, isLoading, error } = useGetPaper(id ?? null)
  const { data: statusData } = useGetPaperStatus(id ?? null)

  const paper = data?.paper ?? null
  const status = statusData ?? null

  const handleDownload = () => {
    if (!id) return
    window.open(`/api/v1/papers/${id}/download`, '_blank')
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (error || !paper) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <AlertCircle className="h-12 w-12 text-muted-foreground mb-4" />
        <h1 className="text-2xl font-semibold mb-2">Paper Not Found</h1>
        <Button onClick={() => navigate('/papers')}>
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to Papers
        </Button>
      </div>
    )
  }

  // Show processing/pending state
  if (paper.status === 'pending' || paper.status === 'processing') {
    return (
      <div className="max-w-4xl mx-auto py-6 px-4">
        <Button variant="ghost" onClick={() => navigate('/papers')} className="mb-6">
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to Papers
        </Button>
        <Card>
          <CardContent className="p-8 text-center">
            <Loader2 className="h-12 w-12 animate-spin mx-auto mb-4 text-primary" />
            <h2 className="text-xl font-semibold mb-2">Converting Paper...</h2>
            <p className="text-muted-foreground mb-4">{paper.original_filename}</p>
            {status && (
              <div className="w-full bg-muted rounded-full h-2">
                <div
                  className="bg-primary h-2 rounded-full transition-all"
                  style={{ width: `${status.progress}%` }}
                />
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    )
  }

  // Show failed state
  if (paper.status === 'failed') {
    return (
      <div className="max-w-4xl mx-auto py-6 px-4">
        <Button variant="ghost" onClick={() => navigate('/papers')} className="mb-6">
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to Papers
        </Button>
        <Card>
          <CardContent className="p-8 text-center">
            <AlertCircle className="h-12 w-12 mx-auto mb-4 text-destructive" />
            <h2 className="text-xl font-semibold mb-2">Conversion Failed</h2>
            <p className="text-muted-foreground mb-4">{paper.error || 'Unknown error'}</p>
            <Button onClick={() => navigate('/papers')}>
              Back to Papers
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  // Show completed paper
  return (
    <div className="max-w-4xl mx-auto py-6 px-4">
      <Button variant="ghost" onClick={() => navigate('/papers')} className="mb-6">
        <ArrowLeft className="h-4 w-4 mr-2" />
        Back to Papers
      </Button>

      {/* Metadata */}
      <Card className="mb-6">
        <CardContent className="p-6">
          <PaperMeta paper={paper} onDownload={handleDownload} />
        </CardContent>
      </Card>

      {/* Markdown Content */}
      {paper.markdown_content ? (
        <Card>
          <CardContent className="p-6">
            <MarkdownRenderer content={paper.markdown_content} />
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent className="p-6 text-center text-muted-foreground">
            No content available
          </CardContent>
        </Card>
      )}
    </div>
  )
}
```

- [ ] **Step 3: 前端编译检查**

Run: `cd web && npx tsc --noEmit`
Expected: 无错误

- [ ] **Step 4: Commit**

```bash
git add web/src/components/papers/PaperMeta.tsx web/src/pages/papers/PaperViewPage.tsx
git commit -m "feat(papers): add PaperViewPage with metadata display and Markdown rendering"
```

---

## Task 13: 前端 — 路由 + 侧边栏集成

**Files:**
- Modify: `web/src/App.tsx`
- Modify: `web/src/components/feed/Sidebar.tsx`

- [ ] **Step 1: 在 App.tsx 中添加论文路由**

在 import 区域添加：
```tsx
import PapersPage from '@/pages/papers/PapersPage'
import PaperViewPage from '@/pages/papers/PaperViewPage'
```

在 Protected routes 区域（`/items/:id` 路由之后）添加：
```tsx
<Route
  path="/papers"
  element={
    <ProtectedRoute>
      <PapersPage />
    </ProtectedRoute>
  }
/>
<Route
  path="/papers/:id"
  element={
    <ProtectedRoute>
      <PaperViewPage />
    </ProtectedRoute>
  }
/>
```

- [ ] **Step 2: 在 Sidebar.tsx 中添加论文库入口**

在 import 区域添加 `BookOpen` icon:
```tsx
import { Home, Star, Rss, Menu, Calendar, BookOpen } from 'lucide-react'
```

在 `SidebarContent` 组件的导航区域（filterConfig map 之后、分割线之前）添加：
```tsx
{/* Divider */}
<div className="border-t my-2" />

{/* Papers section */}
<Button
  variant="ghost"
  className="w-full justify-start"
  onClick={() => window.location.href = '/papers'}
>
  <BookOpen className="w-4 h-4 mr-2" />
  Papers
</Button>
```

- [ ] **Step 3: 前端编译和测试**

Run: `cd web && npx tsc --noEmit`
Expected: 无错误

Run: `cd web && npm test`
Expected: 全部 PASS

- [ ] **Step 4: Commit**

```bash
git add web/src/App.tsx web/src/components/feed/Sidebar.tsx
git commit -m "feat(papers): add paper routes and sidebar navigation entry"
```

---

## Task 14: 端到端验证

**Files:**
- 无新增文件

- [ ] **Step 1: 运行全部后端测试**

Run: `go test ./internal/... -v -count=1`
Expected: 全部 PASS

- [ ] **Step 2: 运行全部前端测试**

Run: `cd web && npm test`
Expected: 全部 PASS

- [ ] **Step 3: 确认编译通过**

Run: `go build ./cmd/server/...`
Expected: 成功

Run: `cd web && npm run build`
Expected: 成功

- [ ] **Step 4: 最终 Commit（如有遗漏修复）**

```bash
git add -A
git commit -m "chore(papers): end-to-end verification and fixes"
```

---

## Self-Review 检查清单

### Spec 覆盖度

| Spec 需求 | 对应 Task |
|-----------|-----------|
| Paper/PaperTag/PaperCollection 数据模型 | Task 2 |
| MinerU + LLM 转换流水线 | Task 9 (Python) |
| gRPC 接口定义 (Convert + ExtractMetadata) | Task 1 |
| gRPC Client 封装 | Task 5 |
| PDF 文件上传 + 磁盘存储 | Task 6 (Service) + Task 7 (Handler) |
| 异步转换 (goroutine) | Task 6 |
| 状态轮询 (status endpoint) | Task 7 + Task 12 |
| 元数据更新 (PUT) | Task 7 |
| 标签管理 | Task 4 + Task 6 + Task 7 |
| 搜索筛选 (q/year/tag/status) | Task 4 + Task 7 + Task 11 |
| 降级策略 (gRPC 不可用/LLM 失败) | Task 6 (failPaper + metadata fallback) |
| 论文列表页 | Task 11 |
| 论文详情页 + Markdown 渲染 | Task 12 |
| 上传模态框 | Task 11 |
| 侧边栏论文入口 | Task 13 |
| 路由 /papers, /papers/:id | Task 13 |
| PaperConfig 环境变量 | Task 3 |
| 重试失败转换 | Task 6 + Task 7 |
| 下载原始 PDF | Task 7 |
| 删除论文（含 PDF 文件） | Task 6 + Task 7 |

### Placeholder 扫描

- [ ] 无 TBD / TODO / "implement later"
- [ ] 无 "add appropriate error handling"
- [ ] 无 "similar to Task N"
- [ ] 所有代码步骤包含完整代码
- [ ] 所有测试步骤包含具体断言

### 类型一致性

- [ ] PaperStatus 常量在 model/service/handler/前端 一致
- [ ] PaperListOptions 在 interfaces/repository/handler 一致
- [ ] API response 格式在 handler/前端 hooks 一致
