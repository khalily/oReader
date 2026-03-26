# Shiki 语法高亮增强设计规范

> **目标**: 用 Shiki 替换 Prism，增强代码块渲染体验

## 概述

将现有的 `react-syntax-highlighter` (Prism) 替换为 Shiki，实现更好的语言支持和主题切换功能。

## 技术选型

### 为什么选择 Shiki

| 特性 | Prism | Shiki |
|------|-------|-------|
| 语言支持 | ~200 (需手动加载) | 180+ (内置) |
| 语法引擎 | 正则表达式 | TextMate 语法 (VS Code 同款) |
| 主题切换 | 手动管理 | 内置双主题支持 |
| 体积 | 较小 | 较大 (WASM) |
| 渲染质量 | 良好 | 精确 (IDE 级别) |

### 依赖包

```json
{
  "dependencies": {
    "shiki": "^1.0.0",
    "@shikijs/rehype": "^1.0.0"
  }
}
```

移除：
```json
{
  "devDependencies": {
    "@types/react-syntax-highlighter": "^15.5.13"
  },
  "dependencies": {
    "react-syntax-highlighter": "^16.1.1"
  }
}
```

## 架构设计

### 组件结构

```
MarkdownRenderer.tsx
├── ReactMarkdown (核心渲染)
│   ├── remarkGfm (GitHub 风格 Markdown)
│   ├── remarkMath (LaTeX 数学公式)
│   ├── rehypeKatex (KaTeX 渲染)
│   └── rehypeShiki (Shiki 语法高亮) ← 新增
└── CopyButton (复制按钮组件) ← 新增
```

### 主题系统

使用 Shiki 双主题模式，自动跟随系统主题：

```typescript
const shikiOptions = {
  themes: {
    light: 'github-light',
    dark: 'github-dark'
  },
  defaultColor: false // 使用 CSS 变量控制
}
```

CSS 变量切换：
```css
/* Shiki 生成的 HTML 使用 data-theme 属性 */
.shiki {
  background-color: var(--shiki-bg);
  color: var(--shiki-fg);
}

[data-theme='light'] .shiki {
  --shiki-bg: #ffffff;
  --shiki-fg: #24292e;
}

[data-theme='dark'] .shiki {
  --shiki-bg: #0d1117;
  --shiki-fg: #c9d1d9;
}
```

### 复制按钮

位置：代码块右上角内部，半透明背景，hover 时显示

```tsx
<div className="relative group">
  <pre className="shiki">...</pre>
  <button className="absolute top-2 right-2 opacity-0 group-hover:opacity-100">
    <CopyIcon />
  </button>
</div>
```

功能：
- 点击复制代码到剪贴板
- 复制成功显示 ✓ 图标，2 秒后恢复
- 使用 `navigator.clipboard.writeText()`

### 行号显示

使用 CSS counter 实现，始终显示：

```css
.shiki {
  counter-reset: line;
}

.shiki .line::before {
  counter-increment: line;
  content: counter(line);
  display: inline-block;
  width: 2em;
  margin-right: 1em;
  text-align: right;
  color: var(--muted-foreground);
  opacity: 0.5;
}
```

## 实现细节

### MarkdownRenderer 组件修改

```tsx
import { codeToHtml } from 'shiki'
import { rehypeShiki } from '@shikijs/rehype'

// Shiki 配置
const shikiHighlighter = await getHighlighter({
  themes: ['github-light', 'github-dark'],
  langs: ['javascript', 'typescript', 'python', 'go', 'rust', 'jsx', 'tsx', 'bash', 'json', 'yaml', 'markdown', 'css', 'html']
})

// ReactMarkdown 配置
<ReactMarkdown
  remarkPlugins={[remarkGfm, remarkMath]}
  rehypePlugins={[
    rehypeKatex,
    [rehypeShiki, { highlighter: shikiHighlighter, themes: { light: 'github-light', dark: 'github-dark' } }]
  ]}
  components={{
    // 处理 rehypeShiki 渲染后的 pre，添加复制按钮
    pre({ children, ...props }) {
      return (
        <div className="relative group">
          <pre {...props}>{children}</pre>
          <CopyButton code={extractCode(children)} />
        </div>
      )
    }
  }}
>
  {content}
</ReactMarkdown>
```

### CopyButton 组件

```tsx
function CopyButton({ code }: { code: string }) {
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    await navigator.clipboard.writeText(code)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <button
      onClick={handleCopy}
      className="absolute top-2 right-2 p-2 rounded-md
                 bg-black/10 dark:bg-white/10
                 opacity-0 group-hover:opacity-100
                 transition-opacity"
    >
      {copied ? <CheckIcon className="h-4 w-4" /> : <CopyIcon className="h-4 w-4" />}
    </button>
  )
}
```

### 主题切换逻辑

方案：使用 Shiki 内置双主题 CSS 变量机制

```css
/* Shiki 双主题输出的 HTML 结构 */
/* <span style="--shiki-light:color;--shiki-dark:color"> */

/* 在根元素设置当前主题 */
:root {
  color-scheme: light;
}

.dark {
  color-scheme: dark;
}

/* Shiki 自动根据 color-scheme 选择颜色 */
```

## 测试策略

### 单元测试

1. **Shiki 渲染测试**
   - 各语言语法高亮正确性
   - 行号显示
   - 复制按钮交互

2. **主题切换测试**
   - 亮色主题样式
   - 暗色主题样式

### 测试用例更新

更新 `MarkdownRenderer.test.tsx`：
- 测试 Shiki 渲染各语言
- 测试复制按钮存在
- 测试行号存在

## 文件变更

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `web/package.json` | 修改 | 添加 shiki 依赖，移除 react-syntax-highlighter |
| `web/src/components/ui/MarkdownRenderer.tsx` | 重写 | 使用 Shiki + rehype 插件 |
| `web/src/index.css` | 修改 | 添加 Shiki 样式和行号样式 |
| `web/src/components/ui/__tests__/MarkdownRenderer.test.tsx` | 修改 | 更新测试用例 |

## 风险与缓解

| 风险 | 缓解措施 |
|------|----------|
| Shiki WASM 加载慢 | 使用 vite 预打包，首次加载后缓存 |
| 首次渲染慢 | 异步初始化 highlighter，显示 loading |
| 体积增大 | Shiki 按需加载语言包 |

## 验收标准

- [ ] 代码块支持 Rust、Go、TypeScript/JSX、Python 等语言
- [ ] 复制按钮点击后代码成功复制到剪贴板
- [ ] 行号始终显示
- [ ] 主题跟随系统亮色/暗色模式切换
- [ ] 现有测试通过
