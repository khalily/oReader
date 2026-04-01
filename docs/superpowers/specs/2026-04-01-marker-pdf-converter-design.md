# Marker PDF Converter 集成设计

## 背景

当前 PDF 转 Markdown 使用 MinerU + LLM 方案，存在以下问题：
- 公式渲染差（MinerU 提取乱码，LLM 只能猜测修复）
- 表格转换差（HTML 表格转 GFM 不可靠）
- 特殊字符编码问题（需要 `fix_utf8_mojibake` 补丁）
- 图片处理复杂（placeholder → LLM → restore 三步流程）
- 依赖 LLM API（成本高、延迟大）

PoC 测试表明 [Marker](https://github.com/datalab-to/marker) 在学术论文场景下表格、公式质量显著优于当前方案，且不需要 LLM refine。

## 方案

用 Marker 替换 MinerU + LLM refine pipeline，保持 gRPC 接口不变。

### 架构变更

**现有流程**:
```
PDF → MinerU (parse) → 图片占位 → LLM refine (分块) → 图片还原 → Markdown
```

**新流程**:
```
PDF → Marker (convert) → 后处理(清理/图片base64化) → Markdown
```

### 变更范围

1. **`converter.py`** — 核心转换逻辑重写
   - 移除 MinerU 相关代码
   - 移除 LLM refine（`refine_markdown`、`_refine_chunk`、`_split_markdown_chunks`）
   - 保留 `extract_metadata()`（LLM metadata 提取不变）
   - 保留 `fix_utf8_mojibake()`（安全网）
   - 新增 `convert_with_marker()` 函数

2. **`server.py`** — gRPC servicer 更新
   - `Convert()` 使用 Marker 替换 MinerU + LLM
   - 简化图片处理
   - 移除 placeholder 相关函数

3. **`pyproject.toml`** — 依赖更新
   - 移除 `magic-pdf[full]`
   - 添加 `marker-pdf`
   - 移除 `transformers`（Marker 自带）

4. **不变更**
   - `paper.proto` — gRPC 接口不变
   - Go backend — 完全透明
   - `interceptor.py` / `logging_config.py` — 不变

### 关键实现细节

**Marker 转换函数**:
```python
def convert_with_marker(pdf_content: bytes, filename: str) -> tuple[str, list[tuple[str, str]]]:
    """Convert PDF to Markdown using Marker.

    Returns (markdown_text, [(image_filename, base64_data), ...])
    """
```

**图片处理**:
- Marker 输出图片文件到临时目录
- 读取图片并转为 base64 data URL
- 替换 Markdown 中的图片引用

**进度报告**:
- 保持现有的 streaming progress 机制
- Phase 1: converting (0-80%)
- Phase 2: post-processing (80-100%)

## 测试策略

- 单元测试：`convert_with_marker()` 函数、图片 base64 化、清理逻辑
- E2E 测试：使用真实 PDF 验证完整 pipeline
