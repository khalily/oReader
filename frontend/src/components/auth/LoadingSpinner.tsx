import { Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'

type LoadingSpinnerProps = {
  visible?: boolean
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

const sizeClasses = {
  sm: 'h-4 w-4',
  md: 'h-6 w-6',
  lg: 'h-8 w-8',
}

export default function LoadingSpinner({
  visible = true,
  size = 'md',
  className,
}: LoadingSpinnerProps) {
  if (!visible) return null

  return (
    <div
      role="status"
      aria-label="loading"
      className={cn('inline-flex items-center justify-center', className)}
    >
      <Loader2 className={cn('animate-spin', sizeClasses[size])} />
      <span className="sr-only">loading</span>
    </div>
  )
}
