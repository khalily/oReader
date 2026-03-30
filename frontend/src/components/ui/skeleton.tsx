import { cn } from '@/lib/utils'

export interface SkeletonProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: 'default' | 'text' | 'circular' | 'rounded'
}

/**
 * Skeleton component for loading states.
 * Use this to show placeholder content while data is loading.
 */
export function Skeleton({ className, variant = 'default', ...props }: SkeletonProps) {
  const variantClasses = {
    default: 'rounded-md',
    text: 'rounded-sm h-4',
    circular: 'rounded-full',
    rounded: 'rounded-lg',
  }

  return (
    <div
      className={cn(
        'animate-pulse bg-muted',
        variantClasses[variant],
        className
      )}
      {...props}
    />
  )
}

/**
 * Skeleton for a feed card in the sidebar
 */
export function FeedSkeleton({ count = 3 }: { count?: number }) {
  return (
    <div className="space-y-2">
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="flex items-center gap-2 p-2 rounded-md">
          <Skeleton variant="circular" className="h-8 w-8 flex-shrink-0" />
          <div className="flex-1 min-w-0">
            <Skeleton variant="text" className="w-3/4 mb-1" />
            <Skeleton variant="text" className="w-1/2 h-3" />
          </div>
        </div>
      ))}
    </div>
  )
}

/**
 * Skeleton for an article card in the item list
 */
export function ItemCardSkeleton() {
  return (
    <div className="p-4 border rounded-md bg-card">
      <div className="flex items-start gap-3">
        {/* Action buttons */}
        <div className="flex flex-col gap-1 flex-shrink-0">
          <Skeleton variant="circular" className="h-8 w-8" />
          <Skeleton variant="circular" className="h-8 w-8" />
        </div>

        {/* Article content */}
        <div className="min-w-0 flex-1">
          <Skeleton className="h-5 w-3/4 mb-2" />
          <div className="flex items-center gap-2 mb-2">
            <Skeleton variant="text" className="w-16 h-3" />
            <Skeleton variant="text" className="w-16 h-3" />
          </div>
          <Skeleton variant="text" className="w-full mb-1" />
          <Skeleton variant="text" className="w-2/3 h-3" />
        </div>

        {/* External link button */}
        <Skeleton variant="circular" className="h-8 w-8 flex-shrink-0" />
      </div>
    </div>
  )
}

/**
 * Skeleton for a list of article cards
 */
export function ItemListSkeleton({ count = 5 }: { count?: number }) {
  return (
    <div className="space-y-2">
      {Array.from({ length: count }).map((_, i) => (
        <ItemCardSkeleton key={i} />
      ))}
    </div>
  )
}

/**
 * Skeleton for article detail view
 */
export function ItemDetailSkeleton() {
  return (
    <div className="max-w-4xl mx-auto py-6 px-4 space-y-6">
      {/* Header */}
      <div className="space-y-4">
        <Skeleton className="h-8 w-3/4" />
        <div className="flex items-center gap-3">
          <Skeleton variant="circular" className="h-10 w-10" />
          <Skeleton variant="text" className="w-32 h-4" />
        </div>
        <div className="flex gap-2">
          <Skeleton className="h-8 w-20" />
          <Skeleton className="h-8 w-24" />
        </div>
      </div>

      {/* Content */}
      <div className="space-y-3">
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-11/12" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-4/5" />
      </div>

      {/* More content */}
      <div className="space-y-3 pt-4">
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-3/4" />
      </div>
    </div>
  )
}
