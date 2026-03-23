import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import SocialLoginButton from '../SocialLoginButton'

// Mock window.location
const originalLocation = window.location

describe('SocialLoginButton', () => {
  beforeEach(() => {
    // Mock window.location
    Object.defineProperty(window, 'location', {
      value: { href: '' },
      writable: true,
    })
  })

  afterEach(() => {
    // Restore original location
    Object.defineProperty(window, 'location', {
      value: originalLocation,
      writable: true,
    })
  })

  it('should render GitHub login button with correct text', () => {
    render(<SocialLoginButton provider="github" />)

    expect(screen.getByRole('button', { name: /sign in with github/i })).toBeInTheDocument()
  })

  it('should have correct GitHub styling classes', () => {
    render(<SocialLoginButton provider="github" />)

    const button = screen.getByRole('button', { name: /sign in with github/i })
    expect(button).toHaveClass('bg-[#24292F]')
    expect(button).toHaveClass('text-white')
  })

  it('should navigate to GitHub OAuth endpoint on click', async () => {
    const user = userEvent.setup()
    render(<SocialLoginButton provider="github" />)

    const button = screen.getByRole('button', { name: /sign in with github/i })
    await user.click(button)

    expect(window.location.href).toBe('/api/v1/auth/github')
  })

  it('should include GitHub icon', () => {
    render(<SocialLoginButton provider="github" />)

    // Check for SVG element (GitHub icon)
    const button = screen.getByRole('button', { name: /sign in with github/i })
    const svg = button.querySelector('svg')
    expect(svg).toBeInTheDocument()
  })

  it('should apply custom className if provided', () => {
    render(<SocialLoginButton provider="github" className="custom-class" />)

    const button = screen.getByRole('button', { name: /sign in with github/i })
    expect(button).toHaveClass('custom-class')
  })
})
