import { type ReactNode } from 'react'
import { X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

type ErrorMessageProps = {
  message: string | Error | { response?: { data?: { error?: { message?: string } } } } | null
  onDismiss?: () => void
  className?: string
}

export function getErrorMessage(message: ErrorMessageProps['message']): string | null {
  if (!message) return null

  if (typeof message === 'string') {
    return message
  }

  // Handle axios error object
  if (typeof message === 'object' && message.response?.data?.error?.message) {
    return message.response.data.error.message as string
  }

  return null
}

export default function ErrorMessage({ message, onDismiss, className }: ErrorMessageProps) {
  const text = getErrorMessage(message)

  if (!text) return null

  return (
    <div className={cn('flex items-center justify-between gap-2 text-sm text-destructive', className)}>
      <span>{text}</span>
      {onDismiss && (
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="h-4 w-4"
          onClick={onDismiss}
        >
          <X className="h-3 w-3" />
          <span className="sr-only">Close error</span>
        </Button>
      )}
    </div>
  )
}
