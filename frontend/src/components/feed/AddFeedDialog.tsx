import { useState, useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useFeeds } from '@/hooks/useFeeds'
import {
  Dialog,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogContent,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

const feedUrlSchema = z.object({
  feed_url: z
    .string()
    .min(1, 'Feed URL is required')
    .url('Invalid URL format')
    .refine(
      (url) => url.startsWith('http://') || url.startsWith('https://'),
      'Only HTTP and HTTPS URLs are supported'
    ),
})

type FeedUrlForm = z.infer<typeof feedUrlSchema>

interface AddFeedDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
}

export function AddFeedDialog({ open, onOpenChange, onSuccess }: AddFeedDialogProps) {
  const [error, setError] = useState<string | null>(null)
  const { useCreateFeed } = useFeeds()

  const {
    register,
    handleSubmit,
    formState: { errors },
    reset,
    clearErrors,
  } = useForm<FeedUrlForm>({
    resolver: zodResolver(feedUrlSchema),
  })

  const createFeed = useCreateFeed()

  useEffect(() => {
    if (open) {
      clearErrors()
      setError(null)
      reset()
    }
  }, [open, clearErrors, reset])

  const onSubmit = async (data: FeedUrlForm) => {
    setError(null)
    try {
      await createFeed.mutateAsync(data)
      reset()
      onOpenChange(false)
      onSuccess?.()
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to add feed'
      setError(errorMessage)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add Feed</DialogTitle>
          <DialogDescription>
            Subscribe to a new RSS feed by entering its URL below.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)}>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="feed_url">Feed URL</Label>
              <Input
                id="feed_url"
                placeholder="Enter RSS feed URL (e.g., https://example.com/feed.xml)"
                {...register('feed_url')}
                disabled={createFeed.isPending}
              />
              {errors.feed_url && (
                <p className="text-sm text-destructive">{errors.feed_url.message}</p>
              )}
              {error && !errors.feed_url && (
                <p className="text-sm text-destructive">{error}</p>
              )}
            </div>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={createFeed.isPending}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={createFeed.isPending}>
              {createFeed.isPending ? 'Adding...' : 'Add'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
