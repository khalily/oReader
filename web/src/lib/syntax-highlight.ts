// web/src/lib/syntax-highlight.ts
import Prism from 'prismjs'

// Import commonly used language support
// Note: These add ~50KB to bundle. Consider code-splitting if bundle size becomes an issue.
import 'prismjs/components/prism-javascript'
import 'prismjs/components/prism-typescript'
import 'prismjs/components/prism-jsx'
import 'prismjs/components/prism-tsx'
import 'prismjs/components/prism-go'
import 'prismjs/components/prism-python'
import 'prismjs/components/prism-bash'
import 'prismjs/components/prism-json'
import 'prismjs/components/prism-yaml'
import 'prismjs/components/prism-markdown'
import 'prismjs/components/prism-css'
import 'prismjs/components/prism-sql'
import 'prismjs/components/prism-diff'

// Import default theme (light)
import 'prismjs/themes/prism.min.css'

/**
 * Highlight all code blocks within a container element
 */
export function highlightCode(container: HTMLElement): void {
  Prism.highlightAllUnder(container)
}

/**
 * Setup lazy image loading and error handling within a container
 */
export function setupImages(container: HTMLElement): void {
  container.querySelectorAll('img').forEach((img) => {
    // Enable native lazy loading
    img.loading = 'lazy'

    // Handle image load errors gracefully
    img.onerror = () => {
      img.src = '/image-placeholder.svg'
      img.alt = 'Image failed to load'
      img.style.opacity = '0.5'
    }
  })
}

/**
 * Process article content after render
 * Handles code highlighting and image optimization
 */
export function processArticleContent(container: HTMLElement): void {
  if (!container) return

  // Apply code highlighting
  highlightCode(container)

  // Setup image handling
  setupImages(container)
}
