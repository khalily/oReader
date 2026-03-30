import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { DeleteConfirmDialog } from '../DeleteConfirmDialog'

describe('DeleteConfirmDialog', () => {
  const defaultProps = {
    open: true,
    onOpenChange: vi.fn(),
    onConfirm: vi.fn(),
    title: 'Delete Feed',
    description: 'Are you sure you want to delete this feed?',
    itemName: 'Test Feed',
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should render dialog when open', () => {
    render(<DeleteConfirmDialog {...defaultProps} />)

    expect(screen.getByText('Delete Feed')).toBeInTheDocument()
    expect(screen.getByText(/Test Feed/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Delete' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument()
  })

  it('should not render when closed', () => {
    render(<DeleteConfirmDialog {...defaultProps} open={false} />)

    expect(screen.queryByText('Delete Feed')).not.toBeInTheDocument()
  })

  it('should call onOpenChange with false when Cancel is clicked', async () => {
    const onOpenChange = vi.fn()
    render(<DeleteConfirmDialog {...defaultProps} onOpenChange={onOpenChange} />)

    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))

    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it('should call onConfirm when Delete is clicked', async () => {
    const onConfirm = vi.fn().mockResolvedValue(undefined)
    const onOpenChange = vi.fn()
    render(
      <DeleteConfirmDialog {...defaultProps} onConfirm={onConfirm} onOpenChange={onOpenChange} />
    )

    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))

    await waitFor(() => {
      expect(onConfirm).toHaveBeenCalled()
    })
  })

  it('should show loading state during deletion', async () => {
    const onConfirm = vi.fn().mockImplementation(() => new Promise((resolve) => setTimeout(resolve, 100)))
    render(<DeleteConfirmDialog {...defaultProps} onConfirm={onConfirm} />)

    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))

    expect(screen.getByText('Deleting...')).toBeInTheDocument()
  })

  it('should disable buttons during deletion', async () => {
    const onConfirm = vi.fn().mockImplementation(() => new Promise((resolve) => setTimeout(resolve, 100)))
    render(<DeleteConfirmDialog {...defaultProps} onConfirm={onConfirm} />)

    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))

    const cancelButton = screen.getByRole('button', { name: 'Cancel' })
    const deleteButton = screen.getByRole('button', { name: 'Deleting...' })

    expect(cancelButton).toBeDisabled()
    expect(deleteButton).toBeDisabled()
  })

  it('should close dialog after successful deletion', async () => {
    const onConfirm = vi.fn().mockResolvedValue(undefined)
    const onOpenChange = vi.fn()
    render(
      <DeleteConfirmDialog {...defaultProps} onConfirm={onConfirm} onOpenChange={onOpenChange} />
    )

    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))

    await waitFor(() => {
      expect(onOpenChange).toHaveBeenCalledWith(false)
    })
  })
})
