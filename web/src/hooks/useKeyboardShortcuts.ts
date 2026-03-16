import { useEffect, useCallback } from 'react'

export interface ShortcutConfig {
  key: string
  description: string
  action: () => void
  enabled?: boolean
  preventDefault?: boolean
}

interface UseKeyboardShortcutsOptions {
  enabled?: boolean
  shortcuts: ShortcutConfig[]
}

/**
 * Hook for handling keyboard shortcuts
 *
 * @example
 * ```tsx
 * useKeyboardShortcuts({
 *   enabled: true,
 *   shortcuts: [
 *     { key: 'j', description: 'Next article', action: () => navigateNext() },
 *     { key: 'k', description: 'Previous article', action: () => navigatePrev() },
 *   ]
 * })
 * ```
 */
export function useKeyboardShortcuts({
  enabled = true,
  shortcuts,
}: UseKeyboardShortcutsOptions) {
  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (!enabled) return

      // Ignore if user is typing in an input
      const target = event.target as HTMLElement
      if (
        target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.isContentEditable
      ) {
        return
      }

      // Find matching shortcut
      const shortcut = shortcuts.find((s) => {
        if (!s.enabled && s.enabled !== false) return true // default enabled
        return s.enabled
      })

      if (!shortcut) return

      // Check for key match (case-insensitive)
      if (event.key.toLowerCase() === shortcut.key.toLowerCase()) {
        if (shortcut.preventDefault !== false) {
          event.preventDefault()
        }
        shortcut.action()
      }
    },
    [enabled, shortcuts]
  )

  useEffect(() => {
    if (enabled) {
      window.addEventListener('keydown', handleKeyDown)
      return () => window.removeEventListener('keydown', handleKeyDown)
    }
  }, [enabled, handleKeyDown])
}

/**
 * Predefined shortcut descriptions for the help modal
 */
export const DEFAULT_SHORTCUTS = [
  { key: 'j', description: 'Go to next article' },
  { key: 'k', description: 'Go to previous article' },
  { key: 'Enter', description: 'Open selected article' },
  { key: 's', description: 'Star/unstar article' },
  { key: 'r', description: 'Mark as read/unread' },
  { key: 'n', description: 'Mark all as read' },
  { key: 'f', description: 'Focus search' },
  { key: '?', description: 'Show keyboard shortcuts' },
  { key: 'Escape', description: 'Close modal/drawer' },
] as const
