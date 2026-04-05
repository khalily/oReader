import { useState, useEffect, useRef, useCallback } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Rss, Upload, FileText, ArrowLeft, Loader2, X, CheckCircle, AlertCircle } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useFeeds } from '@/hooks/useFeeds'
import { useOPML } from '@/hooks/useOPML'
import { useToast } from '@/components/ui/toast'
import apiClient from '@/lib/api/axios'

type DialogView = 'menu' | 'add-feed' | 'import-feed' | 'import-paper'

interface AddDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess: () => void
}

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

const MAX_OPML_SIZE = 1024 * 1024 // 1MB

type ImportPhase = 'upload' | 'progress' | 'done'

export function AddDialog({ open, onOpenChange, onSuccess }: AddDialogProps) {
  const [view, setView] = useState<DialogView>('menu')
  const toast = useToast()

  // --- Add Feed state ---
  const [feedError, setFeedError] = useState<string | null>(null)
  const { useCreateFeed } = useFeeds()
  const createFeed = useCreateFeed()
  const {
    register,
    handleSubmit,
    formState: { errors: feedErrors },
    reset: resetFeedForm,
    clearErrors: clearFeedErrors,
  } = useForm<FeedUrlForm>({
    resolver: zodResolver(feedUrlSchema),
  })

  // --- Import Feed (OPML) state ---
  const [importPhase, setImportPhase] = useState<ImportPhase>('upload')
  const [opmlFile, setOpmlFile] = useState<File | null>(null)
  const [opmlDragActive, setOpmlDragActive] = useState(false)
  const [opmlError, setOpmlError] = useState<string | null>(null)
  const [opmlJobId, setOpmlJobId] = useState<string | null>(null)
  const opmlInputRef = useRef<HTMLInputElement>(null)
  const { useImportOpml, useGetImportStatus } = useOPML()
  const importMutation = useImportOpml()
  const { data: opmlJobStatus } = useGetImportStatus(
    importPhase === 'progress' ? opmlJobId : null
  )

  // --- Import Paper state ---
  const [paperFile, setPaperFile] = useState<File | null>(null)
  const [paperDragActive, setPaperDragActive] = useState(false)
  const [paperUploading, setPaperUploading] = useState(false)
  const [paperError, setPaperError] = useState<string | null>(null)
  const paperInputRef = useRef<HTMLInputElement>(null)

  // Reset all state when dialog opens
  useEffect(() => {
    if (open) {
      setView('menu')
      setFeedError(null)
      resetFeedForm()
      clearFeedErrors()
      setImportPhase('upload')
      setOpmlFile(null)
      setOpmlDragActive(false)
      setOpmlError(null)
      setOpmlJobId(null)
      setPaperFile(null)
      setPaperDragActive(false)
      setPaperUploading(false)
      setPaperError(null)
    }
  }, [open, resetFeedForm, clearFeedErrors])

  // --- Add Feed handlers ---
  const onFeedSubmit = async (data: FeedUrlForm) => {
    setFeedError(null)
    try {
      await createFeed.mutateAsync(data)
      resetFeedForm()
      onOpenChange(false)
      onSuccess()
      toast.showSuccess('Feed added successfully')
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to add feed'
      setFeedError(errorMessage)
      toast.showError(errorMessage)
    }
  }

  // --- OPML Import handlers ---
  const handleOpmlDrag = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setOpmlDragActive(true)
    } else if (e.type === 'dragleave') {
      setOpmlDragActive(false)
    }
  }, [])

  const handleOpmlDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setOpmlDragActive(false)

    const droppedFile = e.dataTransfer.files?.[0]
    if (
      droppedFile &&
      (droppedFile.type === 'text/xml' ||
        droppedFile.name.endsWith('.xml') ||
        droppedFile.name.endsWith('.opml'))
    ) {
      if (droppedFile.size > MAX_OPML_SIZE) {
        setOpmlError('File must be smaller than 1MB')
        return
      }
      setOpmlFile(droppedFile)
      setOpmlError(null)
    } else {
      setOpmlError('Please upload an OPML file (.xml)')
    }
  }, [])

  const handleOpmlFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const selected = e.target.files?.[0]
    if (selected) {
      if (selected.size > MAX_OPML_SIZE) {
        setOpmlError('File must be smaller than 1MB')
        return
      }
      setOpmlFile(selected)
      setOpmlError(null)
    }
  }

  const handleOpmlImport = async () => {
    if (!opmlFile) return

    try {
      setOpmlError(null)
      const result = await importMutation.mutateAsync(opmlFile)
      setOpmlJobId(result.job_id)
      setImportPhase('progress')
    } catch (err: unknown) {
      const msg = (err as Error)?.message || 'Import failed'
      setOpmlError(msg)
      toast.showError(msg)
    }
  }

  const handleOpmlClose = () => {
    if (importPhase === 'done') {
      onSuccess()
    }
    setImportPhase('upload')
    setOpmlFile(null)
    setOpmlError(null)
    setOpmlJobId(null)
  }

  // --- Paper Upload handlers ---
  const handlePaperDrag = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setPaperDragActive(true)
    } else if (e.type === 'dragleave') {
      setPaperDragActive(false)
    }
  }, [])

  const handlePaperDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setPaperDragActive(false)

    const droppedFile = e.dataTransfer.files?.[0]
    if (droppedFile && droppedFile.type === 'application/pdf') {
      setPaperFile(droppedFile)
      setPaperError(null)
    }
  }, [])

  const handlePaperFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const selected = e.target.files?.[0]
    if (selected) {
      setPaperFile(selected)
      setPaperError(null)
    }
  }

  const handlePaperUpload = async () => {
    if (!paperFile || paperUploading) return

    setPaperUploading(true)
    setPaperError(null)
    try {
      const formData = new FormData()
      formData.append('file', paperFile)
      await apiClient.post('/papers/upload', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })

      setPaperFile(null)
      onOpenChange(false)
      onSuccess()
      toast.showSuccess('Paper uploaded successfully')
    } catch (err: unknown) {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const msg = (err as any)?.response?.data?.message || (err as Error)?.message || 'Upload failed'
      setPaperError(msg)
      toast.showError(msg)
    } finally {
      setPaperUploading(false)
    }
  }

  // Handle dialog close — prevent closing during upload
  const handleDialogOpenChange = (isOpen: boolean) => {
    if (!isOpen && paperUploading) return
    if (!isOpen && importPhase === 'progress' && opmlJobStatus?.status !== 'completed' && opmlJobStatus?.status !== 'failed') return
    onOpenChange(isOpen)
  }

  // Back to menu
  const goBackToMenu = () => {
    handleOpmlClose()
    setPaperFile(null)
    setPaperDragActive(false)
    setPaperUploading(false)
    setPaperError(null)
    setView('menu')
  }

  // --- OPML job status ---
  const isOpmlJobDone =
    opmlJobStatus?.status === 'completed' || opmlJobStatus?.status === 'failed'

  // --- Menu View ---
  if (view === 'menu') {
    return (
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Add New</DialogTitle>
            <DialogDescription>
              Choose what you'd like to add.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-3 py-2">
            <button
              className="flex items-center gap-4 rounded-lg border p-4 text-left transition-colors hover:bg-accent"
              onClick={() => setView('add-feed')}
            >
              <Rss className="h-8 w-8 text-primary" />
              <div>
                <p className="font-medium">Add Feed</p>
                <p className="text-sm text-muted-foreground">
                  Subscribe to a new RSS feed by URL
                </p>
              </div>
            </button>
            <button
              className="flex items-center gap-4 rounded-lg border p-4 text-left transition-colors hover:bg-accent"
              onClick={() => setView('import-feed')}
            >
              <Upload className="h-8 w-8 text-primary" />
              <div>
                <p className="font-medium">Import Feeds</p>
                <p className="text-sm text-muted-foreground">
                  Import RSS subscriptions from an OPML file
                </p>
              </div>
            </button>
            <button
              className="flex items-center gap-4 rounded-lg border p-4 text-left transition-colors hover:bg-accent"
              onClick={() => setView('import-paper')}
            >
              <FileText className="h-8 w-8 text-primary" />
              <div>
                <p className="font-medium">Import Paper</p>
                <p className="text-sm text-muted-foreground">
                  Upload an academic paper in PDF format
                </p>
              </div>
            </button>
          </div>
        </DialogContent>
      </Dialog>
    )
  }

  // --- Add Feed View ---
  if (view === 'add-feed') {
    return (
      <Dialog open={open} onOpenChange={handleDialogOpenChange}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <div className="flex items-center gap-2">
              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8"
                onClick={goBackToMenu}
              >
                <ArrowLeft className="h-4 w-4" />
              </Button>
              <DialogTitle>Add Feed</DialogTitle>
            </div>
            <DialogDescription>
              Subscribe to a new RSS feed by entering its URL.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleSubmit(onFeedSubmit)}>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="add-dialog-feed-url">Feed URL</Label>
                <Input
                  id="add-dialog-feed-url"
                  placeholder="e.g., https://example.com/feed.xml"
                  {...register('feed_url')}
                  disabled={createFeed.isPending}
                />
                {feedErrors.feed_url && (
                  <p className="text-sm text-destructive">{feedErrors.feed_url.message}</p>
                )}
                {feedError && !feedErrors.feed_url && (
                  <p className="text-sm text-destructive">{feedError}</p>
                )}
              </div>
            </div>
            <div className="flex flex-col-reverse sm:flex-row sm:justify-end sm:space-x-2 mt-4">
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
                disabled={createFeed.isPending}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={createFeed.isPending}>
                {createFeed.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Adding...
                  </>
                ) : (
                  'Add'
                )}
              </Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>
    )
  }

  // --- Import Feed (OPML) View ---
  if (view === 'import-feed') {
    return (
      <Dialog open={open} onOpenChange={handleDialogOpenChange}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <div className="flex items-center gap-2">
              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8"
                onClick={goBackToMenu}
              >
                <ArrowLeft className="h-4 w-4" />
              </Button>
              <DialogTitle>Import Feeds</DialogTitle>
            </div>
            <DialogDescription>
              Import RSS subscriptions from an OPML file.
            </DialogDescription>
          </DialogHeader>

          {importPhase === 'upload' && (
            <>
              <div
                className={`border-2 border-dashed rounded-lg p-8 text-center transition-colors cursor-pointer ${
                  opmlDragActive
                    ? 'border-primary bg-primary/5'
                    : 'border-muted-foreground/25'
                }`}
                onDragEnter={handleOpmlDrag}
                onDragLeave={handleOpmlDrag}
                onDragOver={handleOpmlDrag}
                onDrop={handleOpmlDrop}
                onClick={() => opmlInputRef.current?.click()}
              >
                <input
                  ref={opmlInputRef}
                  type="file"
                  accept=".xml,.opml"
                  className="hidden"
                  onChange={handleOpmlFileChange}
                />
                {opmlFile ? (
                  <div className="flex flex-col items-center gap-2">
                    <FileText className="h-10 w-10 text-muted-foreground" />
                    <p className="text-sm font-medium">{opmlFile.name}</p>
                    <p className="text-xs text-muted-foreground">
                      {(opmlFile.size / 1024).toFixed(1)} KB
                    </p>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={(e) => {
                        e.stopPropagation()
                        setOpmlFile(null)
                      }}
                    >
                      <X className="h-3 w-3 mr-1" /> Remove
                    </Button>
                  </div>
                ) : (
                  <div className="flex flex-col items-center gap-2">
                    <Upload className="h-10 w-10 text-muted-foreground" />
                    <p className="text-sm text-muted-foreground">
                      Drag and drop an OPML file, or click to browse
                    </p>
                    <p className="text-xs text-muted-foreground">Max 1MB</p>
                  </div>
                )}
              </div>

              <Button
                className="w-full"
                disabled={!opmlFile || importMutation.isPending}
                onClick={handleOpmlImport}
              >
                {importMutation.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Importing...
                  </>
                ) : (
                  <>
                    <Upload className="h-4 w-4 mr-2" />
                    Import
                  </>
                )}
              </Button>

              {opmlError && (
                <p className="text-sm text-destructive text-center">{opmlError}</p>
              )}
            </>
          )}

          {importPhase === 'progress' && opmlJobStatus && !isOpmlJobDone && (
            <div className="py-4">
              <div className="flex items-center gap-3 mb-4">
                <Loader2 className="h-5 w-5 animate-spin text-primary" />
                <span className="text-sm font-medium">
                  {opmlJobStatus.status === 'pending'
                    ? 'Starting import...'
                    : 'Importing feeds...'}
                </span>
              </div>
              <div className="w-full bg-muted rounded-full h-2 mb-2">
                <div
                  className="bg-primary h-2 rounded-full transition-all"
                  style={{ width: `${opmlJobStatus.progress}%` }}
                />
              </div>
              <p className="text-xs text-muted-foreground text-center">
                {opmlJobStatus.processed} / {opmlJobStatus.total} feeds processed
                {opmlJobStatus.failed > 0 &&
                  ` (${opmlJobStatus.failed} failed)`}
              </p>
            </div>
          )}

          {importPhase === 'progress' && isOpmlJobDone && (
            <div className="py-4">
              {opmlJobStatus!.status === 'completed' ? (
                <div className="flex flex-col items-center gap-3">
                  <CheckCircle className="h-10 w-10 text-green-500" />
                  <div className="text-center">
                    <p className="font-medium">Import Complete</p>
                    <p className="text-sm text-muted-foreground">
                      {opmlJobStatus!.processed} feeds imported
                      {opmlJobStatus!.failed > 0 &&
                        `, ${opmlJobStatus!.failed} failed`}
                    </p>
                  </div>
                </div>
              ) : (
                <div className="flex flex-col items-center gap-3">
                  <AlertCircle className="h-10 w-10 text-destructive" />
                  <div className="text-center">
                    <p className="font-medium">Import Failed</p>
                    <p className="text-sm text-muted-foreground">
                      {opmlJobStatus!.error || 'Unknown error'}
                    </p>
                  </div>
                </div>
              )}
              <Button
                className="w-full mt-4"
                onClick={() => {
                  handleOpmlClose()
                  onOpenChange(false)
                }}
              >
                {opmlJobStatus!.status === 'completed' ? 'Done' : 'Close'}
              </Button>
            </div>
          )}
        </DialogContent>
      </Dialog>
    )
  }

  // --- Import Paper View ---
  if (view === 'import-paper') {
    return (
      <Dialog open={open} onOpenChange={handleDialogOpenChange}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <div className="flex items-center gap-2">
              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8"
                onClick={goBackToMenu}
              >
                <ArrowLeft className="h-4 w-4" />
              </Button>
              <DialogTitle>Upload Paper</DialogTitle>
            </div>
            <DialogDescription>
              Upload an academic paper in PDF format for conversion.
            </DialogDescription>
          </DialogHeader>

          <div
            className={`border-2 border-dashed rounded-lg p-8 text-center transition-colors cursor-pointer ${
              paperDragActive
                ? 'border-primary bg-primary/5'
                : 'border-muted-foreground/25'
            }`}
            onDragEnter={handlePaperDrag}
            onDragLeave={handlePaperDrag}
            onDragOver={handlePaperDrag}
            onDrop={handlePaperDrop}
            onClick={() => paperInputRef.current?.click()}
          >
            <input
              ref={paperInputRef}
              type="file"
              accept=".pdf"
              className="hidden"
              onChange={handlePaperFileChange}
            />
            {paperFile ? (
              <div className="flex flex-col items-center gap-2">
                <FileText className="h-10 w-10 text-muted-foreground" />
                <p className="text-sm font-medium">{paperFile.name}</p>
                <p className="text-xs text-muted-foreground">
                  {(paperFile.size / 1024 / 1024).toFixed(2)} MB
                </p>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={(e) => {
                    e.stopPropagation()
                    setPaperFile(null)
                  }}
                >
                  <X className="h-3 w-3 mr-1" /> Remove
                </Button>
              </div>
            ) : (
              <div className="flex flex-col items-center gap-2">
                <Upload className="h-10 w-10 text-muted-foreground" />
                <p className="text-sm text-muted-foreground">
                  Drag and drop a PDF here, or click to browse
                </p>
                <p className="text-xs text-muted-foreground">
                  Supports academic papers in PDF format
                </p>
              </div>
            )}
          </div>

          <Button
            className="w-full"
            disabled={!paperFile || paperUploading}
            onClick={handlePaperUpload}
          >
            {paperUploading ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                Uploading...
              </>
            ) : (
              <>
                <Upload className="h-4 w-4 mr-2" />
                Upload
              </>
            )}
          </Button>

          {paperError && (
            <p className="text-sm text-destructive text-center">{paperError}</p>
          )}
        </DialogContent>
      </Dialog>
    )
  }

  // Fallback (should not reach here)
  return null
}
