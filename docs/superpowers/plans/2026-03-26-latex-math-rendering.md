# LaTeX 数学公式渲染实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 MarkdownRenderer 组件添加 LaTeX 数学公式渲染支持，支持行内公式 `$...$` 和块级公式 `$$...$$`

**Architecture:** 使用 `remark-math` 解析数学公式语法，`rehype-katex` 将 LaTeX 渲染为 HTML。KaTeX 是一个快速的 TeX 数学渲染库，比 MathJax 性能更好，适合在浏览器中使用。

**Tech Stack:** react-markdown, remark-math, rehype-katex, katex

---

## File Structure

| File | Action | Purpose |
|------|--------|---------|
| `web/src/components/ui/MarkdownRenderer.tsx` | Modify | 添加 LaTeX 渲染支持 |
| `web/src/components/ui/__tests__/MarkdownRenderer.test.tsx` | Modify | 添加 LaTeX 渲染测试 |
| `web/package.json` | Modify | 添加 katex, remark-math, rehype-katex 依赖 |
| `web/src/index.css` | Modify | 添加 KaTeX CSS 样式导入 |

---

### Task 1: 安装依赖

**Files:**
- Modify: `web/package.json`

- [ ] **Step 1: 安装 KaTeX 和相关插件**

```bash
cd web && npm install katex remark-math rehype-katex
```

- [ ] **Step 2: 安装类型定义（如需要）**

```bash
cd web && npm install -D @types/katex
```

- [ ] **Step 3: 验证安装**

Run: `cd web && npm ls katex remark-math rehype-katex`
Expected: 显示已安装的版本

---

### Task 2: 导入 KaTeX 样式

**Files:**
- Modify: `web/src/index.css`

- [ ] **Step 1: 添加 KaTeX CSS 导入**

在 `web/src/index.css` 文件顶部添加：

```css
@import 'katex/dist/katex.min.css';
```

- [ ] **Step 2: 验证样式文件存在**

Run: `ls web/node_modules/katex/dist/katex.min.css`
Expected: 文件存在

---

### Task 3: 编写 LaTeX 渲染测试

**Files:**
- Modify: `web/src/components/ui/__tests__/MarkdownRenderer.test.tsx`

- [ ] **Step 1: 添加所有 LaTeX 测试用例**

在测试文件末尾添加以下测试：

```typescript
  it('renders inline LaTeX math with $...$', () => {
    render(<MarkdownRenderer content="The formula $E = mc^2$ is famous." />)
    // KaTeX renders math in span elements with class katex
    const katexElement = document.querySelector('.katex')
    expect(katexElement).toBeInTheDocument()
    expect(screen.getByText(/E/)).toBeInTheDocument()
  })

  it('renders block LaTeX math with $$...$$', () => {
    render(<MarkdownRenderer content={'$$\\frac{\\partial L}{\\partial w} = \\nabla$$'} />)
    const katexElement = document.querySelector('.katex')
    expect(katexElement).toBeInTheDocument()
  })

  it('renders complex partial derivative formulas', () => {
    const formula = '$$\\frac{\\partial(a \\cdot b)}{\\partial a} = b$$'
    render(<MarkdownRenderer content={formula} />)
    const katexElement = document.querySelector('.katex')
    expect(katexElement).toBeInTheDocument()
  })

  it('renders LaTeX formulas inside GFM tables', () => {
    const tableWithMath = `| Operation | Forward | Local gradients |
|-----------|---------|----------------|
| \`a + b\` | $$a + b$$ | $$\\frac{\\partial}{\\partial a} = 1$$ |`
    render(<MarkdownRenderer content={tableWithMath} />)
    // Should have both table and katex rendered
    expect(screen.getByRole('table')).toBeInTheDocument()
    expect(document.querySelectorAll('.katex').length).toBeGreaterThan(0)
  })
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd web && npm test -- MarkdownRenderer.test.tsx --run`
Expected: FAIL - 因为尚未实现 LaTeX 支持（新测试应该失败）

- [ ] **Step 3: Commit 测试文件**

```bash
git add web/src/components/ui/__tests__/MarkdownRenderer.test.tsx
git commit -m "test: add LaTeX math rendering tests (red)"
```

---

### Task 4: 实现 LaTeX 渲染

**Files:**
- Modify: `web/src/components/ui/MarkdownRenderer.tsx`

- [ ] **Step 1: 添加 remark-math 和 rehype-katex 导入**

在文件顶部添加导入：

```typescript
import remarkMath from 'remark-math'
import rehypeKatex from 'rehype-katex'
```

- [ ] **Step 2: 更新 ReactMarkdown 配置**

将 `ReactMarkdown` 组件配置从：

```typescript
<ReactMarkdown
  remarkPlugins={[remarkGfm]}
  ...
>
```

修改为：

```typescript
<ReactMarkdown
  remarkPlugins={[remarkGfm, remarkMath]}
  rehypePlugins={[rehypeKatex]}
  ...
>
```

- [ ] **Step 3: 运行测试验证通过**

Run: `cd web && npm test -- MarkdownRenderer.test.tsx --run`
Expected: PASS - 所有 LaTeX 测试通过

- [ ] **Step 4: 运行全部测试确保无回归**

Run: `cd web && npm test -- --run`
Expected: PASS - 所有测试通过

- [ ] **Step 5: Commit 实现文件**

```bash
git add web/src/components/ui/MarkdownRenderer.tsx web/src/index.css web/package.json
git commit -m "feat: add LaTeX math rendering support with KaTeX (green)"
```

---

### Task 5: 集成测试和文档

**Files:**
- Modify: `web/package.json` (package-lock.json 会自动更新)

- [ ] **Step 1: 启动开发服务器进行手动测试**

Run: `cd web && npm run dev`

手动测试以下内容：
1. 行内公式：`The formula $E = mc^2$ is famous.`
2. 块级公式：`$$\frac{\partial L}{\partial w} = \nabla$$`
3. 表格中的公式（如用户示例中的数学表格）

- [ ] **Step 2: 更新 CLAUDE.md 文档（可选）**

如果需要，在 CLAUDE.md 中记录 LaTeX 支持信息。

- [ ] **Step 3: 最终 commit**

```bash
git add web/package-lock.json
git commit -m "chore: update package-lock.json for KaTeX dependencies"
```

---

## 参考示例

用户提供的参考格式支持以下 LaTeX 语法：

```markdown
| Operation | Forward | Local gradients |
|-----------|---------|----------------|
| `a + b` | $$a + b$$ | $$\frac{\partial}{\partial a} = 1, \quad \frac{\partial}{\partial b} = 1$$ |
| `a * b` | $$a \cdot b$$ | $$\frac{\partial}{\partial a} = b, \quad \frac{\partial}{\partial b} = a$$ |
```

这将渲染为带有数学公式的表格。

## 注意事项

1. **转义字符**: 在 JavaScript 字符串中，`\` 需要写成 `\\`，例如 `\\frac` 而不是 `\frac`
2. **性能**: KaTeX 是最快的数学渲染库，适合实时渲染
3. **兼容性**: 保持与现有 Markdown 功能（GFM、代码高亮）的兼容性
