# PDF 论文导入功能设计

> 日期：2026-03-27
> 状态：已确认

## 概述

为 oReader 增加 PDF 论文导入功能。用户上传学术论文 PDF，系统通过 MinerU + LLM API 转换为高质量 Markdown 并提取元数据，在 Web 页面上展示和管理。

### 目标

- 支持学术论文（arXiv、IEEE、ACM 等）的 PDF 导入
- 准确保留公式（LaTeX）、双栏布局、表格、图片
- 提取论文元数据（标题、作者、摘要、关键词、年份、DOI）
- 提供分类（标签 + 集合）和搜索功能

### 范围

- 包含：上传、转换、元数据提取、分类搜索、展示
- 不包含：笔记标注、URL 导入、OCR 扫描件支持

## 整体架构

```
┌──────────────┐     ┌──────────────┐     ┌─────────────────────┐
│   Frontend   │────▶│  Go Backend  │────▶│  Python gRPC 服务    │
│  (React)     │◀────│  (API 层)    │◀────│  (MinerU + LLM API) │
└──────────────┘     └──────────────┘     └─────────────────────┘
       HTTP REST           gRPC (protobuf)
```

三层架构：

1. **Frontend (React)**：上传 UI、论文列表、论文详情页（复用 MarkdownRenderer）
2. **Go Backend**：文件接收、异步任务调度、数据存储、API 层
3. **Python gRPC 服务**：MinerU 提取 + LLM 精修 + 元数据提取

### 技术选型

| 组件 | 选择 | 理由 |
|------|------|------|
| PDF 解析 | MinerU (OpenDataLab) | 专为学术论文设计，支持公式→LaTeX、双栏、表格 |
| 格式精修 + 元数据提取 | OpenAI 兼容 LLM API | 一次调用同时修正格式和提取元数据 |
| 内部通信 | gRPC (protobuf) | 强类型、二进制高效、流式进度推送 |
| PDF 文件存储 | 本地文件系统 | 简单可靠，按用户/日期组织 |

## 数据模型

### 新增 4 张表

**papers（核心表）**：

```go
type Paper struct {
    Base
    UserID           string         `gorm:"type:varchar(36);not null;index" json:"user_id"`
    Title            string         `gorm:"type:varchar(500)" json:"title"`
    Authors          string         `gorm:"type:json" json:"authors"`          // JSON 数组
    Abstract         string         `gorm:"type:text" json:"abstract"`
    Keywords         string         `gorm:"type:json" json:"keywords"`         // JSON 数组
    PublishedYear    string         `gorm:"type:varchar(10)" json:"published_year"`
    DOI              string         `gorm:"type:varchar(200)" json:"doi"`
    PDFPath          string         `gorm:"type:varchar(500)" json:"pdf_path"`
    PDFSize          int64          `json:"pdf_size"`
    MarkdownContent  string         `gorm:"type:longtext" json:"markdown_content"`
    CoverImage       string         `gorm:"type:varchar(500)" json:"cover_image"`
    OriginalFilename string         `gorm:"type:varchar(255)" json:"original_filename"`
    Status           string         `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
    Error            string         `gorm:"type:text" json:"error,omitempty"`
    User             *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
```

**paper_tags（标签表）**：

```go
type PaperTag struct {
    Base
    PaperID string `gorm:"type:varchar(36);not null;index" json:"paper_id"`
    Tag     string `gorm:"type:varchar(100);not null;index" json:"tag"`
    Paper   *Paper `gorm:"foreignKey:PaperID" json:"paper,omitempty"`
}
```

**paper_collections（集合表）**：

```go
type PaperCollection struct {
    Base
    UserID      string `gorm:"type:varchar(36);not null;index" json:"user_id"`
    Name        string `gorm:"type:varchar(100);not null" json:"name"`
    Description string `gorm:"type:text" json:"description"`
    User        *User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
```

**paper_collection_items（集合-论文关联表）**：

```go
type PaperCollectionItem struct {
    Base
    CollectionID string            `gorm:"type:varchar(36);not null;index" json:"collection_id"`
    PaperID      string            `gorm:"type:varchar(36);not null;index" json:"paper_id"`
    Collection   *PaperCollection  `gorm:"foreignKey:CollectionID" json:"collection,omitempty"`
    Paper        *Paper            `gorm:"foreignKey:PaperID" json:"paper,omitempty"`
}
```

### 设计决策

- authors/keywords 用 JSON 数组而非关联表：论文作者数量有限（通常 < 10），JSON 足够
- PDF 文件按 `uploads/papers/{user_id}/{date}/` 组织
- status 跟踪转换状态：`pending → processing → completed/failed`
- 复用现有 ImportJob 模式，加 `job_type` 字段区分 OPML/PDF 导入

## 后端处理流水线

### 四阶段异步处理

```
Phase 1: 上传     →  Phase 2: MinerU 提取  →  Phase 3: LLM 精修  →  Phase 4: 存储
PDF → 磁盘         gRPC 调用 Python 服务      修正格式 + 提取元数据   写入 DB
创建 Paper 记录    提取 Markdown + LaTeX      (title/authors/...)     status=completed
启动 goroutine     图片单独保存到磁盘
```

### gRPC 接口定义

```protobuf
service PaperConverter {
  rpc Convert(PdfConvertRequest) returns (stream ConvertProgress);
  rpc ExtractMetadata(MetadataRequest) returns (PaperMetadata);
}

message PdfConvertRequest {
  bytes pdf_content = 1;
  string filename = 2;
}

message ConvertProgress {
  string status = 1;      // mining / llm_refining / completed / failed
  int32 progress = 2;     // 0-100
  string markdown = 3;    // 最终结果（status=completed 时填充）
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

### LLM 调用策略

- 将 MinerU 输出整体发送给 LLM（不分页），一次调用完成：
  1. 修正明显格式错误（公式渲染、表格格式、段落分隔）
  2. 提取结构化元数据（标题、作者、摘要、关键词、年份、DOI）
- 典型论文约 5000-15000 字 Markdown，LLM 调用约 10-30k tokens

### 降级策略

| 场景 | 处理方式 |
|------|----------|
| MinerU 失败 | 标记 status=failed，保留 PDF 文件供重试 |
| LLM API 超时/限流 | 指数退避重试 3 次；仍失败则保存 MinerU 原始输出，元数据留空 |
| gRPC 服务不可用 | 任务 pending，定期重试（backoff）；提供手动重试 API |
| PDF 文件损坏 | 标记 failed，提示用户检查文件 |

## 前端页面设计

### 路由

| 路由 | 页面 | 说明 |
|------|------|------|
| `/papers` | PapersPage | 论文列表 + 搜索筛选 |
| `/papers/:id` | PaperViewPage | 论文详情 + Markdown 渲染 |

上传功能以模态框形式实现。

### 侧边栏扩展

在现有 RSS 订阅源下方新增"论文库"区域：
- "全部论文"入口 + 集合列表 + "导入论文"按钮
- 与 RSS 功能用分割线隔开

### 论文列表页 (PapersPage)

- 顶栏：搜索框 + 年份/标签筛选 + 上传按钮
- 列表项：封面缩略图 + 标题 + 作者 + 年份 + 摘要摘要 + 标签
- 三种状态展示：已完成（正常）、转换中（进度条）、失败（重试/删除）

### 论文详情页 (PaperViewPage)

- 顶部元数据区：标题、作者、年份、DOI、来源
- 摘要区：折叠式摘要展示
- 标签栏：自动提取 + 用户可编辑
- 主体：复用 MarkdownRenderer 渲染 Markdown 内容
- 操作栏：下载原始 PDF、编辑元数据

## API 设计

所有端点挂在 `/api/v1/papers` 下，走 Auth + CSRF 中间件。

### 端点列表

| Method | Path | 说明 |
|--------|------|------|
| POST | /api/v1/papers/upload | 上传 PDF，启动异步转换 |
| GET | /api/v1/papers | 论文列表（分页、筛选、搜索） |
| GET | /api/v1/papers/:id | 论文详情（含 Markdown） |
| GET | /api/v1/papers/:id/status | 转换状态轮询 |
| PUT | /api/v1/papers/:id | 更新元数据/标签 |
| POST | /api/v1/papers/:id/retry | 重试失败转换 |
| DELETE | /api/v1/papers/:id | 删除论文（含 PDF 文件） |
| GET | /api/v1/papers/:id/download | 下载原始 PDF |
| GET | /api/v1/papers/tags | 获取所有标签（筛选用） |

### 查询参数 (GET /api/v1/papers)

```
?page=1&per_page=20          // 分页
&q=transformer               // 搜索标题、作者、关键词
&year=2024                   // 按年份筛选
&tag=深度学习                 // 按标签筛选
&status=completed            // 按状态筛选
&sort=created_at&order=desc  // 排序
```

## 测试策略

### Go 后端（TDD）

- **Repository 层**：集成测试，使用真实 SQLite 数据库
- **Service 层**：单元测试，mock gRPC client + mock repository
- **Handler 层**：httptest 测试 API 端点
- 重点覆盖：并发上传、大文件处理、gRPC 超时/重试逻辑

### Python gRPC 服务

- **单元测试**：MinerU 输出解析、LLM prompt 构造、元数据提取
- **集成测试**：使用真实 arXiv 论文 PDF 端到端测试
- 测试样本：单栏/双栏论文、含公式/表格论文

### 前端

- **组件测试**：PaperList、PaperView 渲染测试
- **交互测试**：上传流程（文件选择 → 进度 → 完成/失败）
- **搜索筛选**：参数变化触发列表更新

## 目录结构（新增文件）

```
# Go 后端
internal/model/paper.go                    # Paper/PaperTag/PaperCollection 模型
internal/handler/paper_handler.go          # API handler
internal/service/paper_service.go          # 业务逻辑
internal/repository/paper_repository.go    # 数据访问
internal/infra/grpc/paper_client.go        # gRPC client 封装
migrations/002_papers.up.sql               # 数据库迁移

# Python gRPC 服务
converter/
├── proto/paper.proto                      # protobuf 定义
├── server.py                              # gRPC 服务入口
├── converter.py                           # MinerU + LLM 转换逻辑
├── requirements.txt                       # Python 依赖
└── tests/                                 # 测试

# 前端
web/src/pages/papers/PapersPage.tsx        # 论文列表页
web/src/pages/papers/PaperViewPage.tsx     # 论文详情页
web/src/components/papers/PaperList.tsx    # 论文列表组件
web/src/components/papers/PaperUpload.tsx  # 上传模态框
web/src/components/papers/PaperMeta.tsx    # 元数据头部组件
web/src/lib/api/papers.ts                 # API client
web/src/stores/papersStore.ts             # 状态管理
```
