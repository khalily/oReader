import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useKeyboardShortcuts } from '../useKeyboardShortcuts'

describe('useKeyboardShortcuts', () => {
  const shortcuts = [
    { key: 'j', description: 'Next', action: vi.fn() },
    { key: 'k', description: 'Previous', action: vi.fn() },
    { key: 's', description: 'Star', action: vi.fn(), preventDefault: true },
  ]

  let addEventListenerSpy: jest.SpyInstance
  let removeEventListenerSpy: jest.SpyInstance

  beforeEach(() => {
    addEventListenerSpy = vi.spyOn(window, 'addEventListener')
    removeEventListenerSpy = vi.spyOn(window, 'removeEventListener')
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('should register keydown listener when enabled', () => {
    renderHook(() => useKeyboardShortcuts({ enabled: true, shortcuts }))

    expect(addEventListenerSpy).toHaveBeenCalledWith('keydown', expect.any(Function))
  })

  it('should remove keydown listener on cleanup', () => {
    const { unmount } = renderHook(() =>
      useKeyboardShortcuts({ enabled: true, shortcuts })
    )

    unmount()

    expect(removeEventListenerSpy).toHaveBeenCalledWith('keydown', expect.any(Function))
  })

  it('should not register listener when disabled', () => {
    renderHook(() => useKeyboardShortcuts({ enabled: false, shortcuts }))

    expect(addEventListenerSpy).not.toHaveBeenCalled()
  })

  it('should call action when key is pressed', () => {
    const action = vi.fn()
    const shortcutsWithAction = [{ key: 'a', description: 'Test', action }]

    renderHook(() => useKeyboardShortcuts({ enabled: true, shortcuts: shortcutsWithAction }))

    // Simulate keydown event
    const event = new KeyboardEvent('keydown', { key: 'a' })
    window.dispatchEvent(event)

    expect(action).toHaveBeenCalled()
  })

  it('should ignore shortcuts when typing in input', () => {
    const action = vi.fn()
    const shortcutsWithAction = [{ key: 'a', description: 'Test', action }]

    renderHook(() => useKeyboardShortcuts({ enabled: true, shortcuts: shortcutsWithAction }))

    // Simulate keydown event from input
    const input = document.createElement('input')
    const event = new KeyboardEvent('keydown', { key: 'a' })
    Object.defineProperty(event, 'target', { value: input })

    window.dispatchEvent(event)

    expect(action).not.toHaveBeenCalled()
  })

  it('should ignore shortcuts when typing in textarea', () => {
    const action = vi.fn()
    const shortcutsWithAction = [{ key: 'a', description: 'Test', action }]

    renderHook(() => useKeyboardShortcuts({ enabled: true, shortcuts: shortcutsWithAction }))

    // Simulate keydown event from textarea
    const textarea = document.createElement('textarea')
    const event = new KeyboardEvent('keydown', { key: 'a' })
    Object.defineProperty(event, 'target', { value: textarea })

    window.dispatchEvent(event)

    expect(action).not.toHaveBeenCalled()
  })

  it('should respect enabled flag on individual shortcuts', () => {
    const action = vi.fn()
    const shortcutsWithAction = [
      { key: 'a', description: 'Test', action, enabled: false },
    ]

    renderHook(() => useKeyboardShortcuts({ enabled: true, shortcuts: shortcutsWithAction }))

    const event = new KeyboardEvent('keydown', { key: 'a' })
    window.dispatchEvent(event)

    expect(action).not.toHaveBeenCalled()
  })

  it('should call preventDefault when configured', () => {
    const action = vi.fn()
    const shortcutsWithAction = [
      { key: 'a', description: 'Test', action, preventDefault: true },
    ]

    renderHook(() => useKeyboardShortcuts({ enabled: true, shortcuts: shortcutsWithAction }))

    const event = new KeyboardEvent('keydown', { key: 'a', cancelable: true })
    const preventDefaultSpy = vi.spyOn(event, 'preventDefault')

    window.dispatchEvent(event)

    expect(preventDefaultSpy).toHaveBeenCalled()
  })

  it('should match keys case-insensitively', () => {
    const action = vi.fn()
    const shortcutsWithAction = [{ key: 'J', description: 'Test', action }]

    renderHook(() => useKeyboardShortcuts({ enabled: true, shortcuts: shortcutsWithAction }))

    // Press lowercase 'j'
    const event = new KeyboardEvent('keydown', { key: 'j' })
    window.dispatchEvent(event)

    expect(action).toHaveBeenCalled()
  })

  it('should handle Escape key', () => {
    const action = vi.fn()
    const shortcutsWithAction = [{ key: 'Escape', description: 'Close', action }]

    renderHook(() => useKeyboardShortcuts({ enabled: true, shortcuts: shortcutsWithAction }))

    const event = new KeyboardEvent('keydown', { key: 'Escape' })
    window.dispatchEvent(event)

    expect(action).toHaveBeenCalled()
  })

  it('should handle Enter key', () => {
    const action = vi.fn()
    const shortcutsWithAction = [{ key: 'Enter', description: 'Open', action }]

    renderHook(() => useKeyboardShortcuts({ enabled: true, shortcuts: shortcutsWithAction }))

    const event = new KeyboardEvent('keydown', { key: 'Enter' })
    window.dispatchEvent(event)

    expect(action).toHaveBeenCalled()
  })
})
