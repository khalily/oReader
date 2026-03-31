import { useState, useCallback, useRef } from 'react'
import { Upload, X, FileText, Loader2, CheckCircle, AlertCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { useOPML } from '@/hooks/useOPML'

const MAX_OPML_SIZE = 1024 * 1024 // 1MB

interface OpmlImportDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess: () => void
}

type ImportPhase = 'upload' | 'progress' | 'done'

export function OpmlImportDialog({ open, onOpenChange, onSuccess }: OpmlImportDialogProps) {
  const [phase, setPhase] = useState<ImportPhase>('upload')
  const [file, setFile] = useState<File | null>(null)
  const [dragActive, setDragActive] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [jobId, setJobId] = useState<string | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const { useImportOpml, useGetImportStatus } = useOPML()
  const importMutation = useImportOpml()
  const { data: jobStatus } = useGetImportStatus(
    phase === 'progress' ? jobId : null
  )

  const handleDrag = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true)
    } else if (e.type === 'dragleave') {
      setDragActive(false)
    }
  }, [])

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setDragActive(false)

    const droppedFile = e.dataTransfer.files?.[0]
    if (droppedFile && (droppedFile.type === 'text/xml' || droppedFile.name.endsWith('.xml') || droppedFile.name.endsWith('.opml'))) {
      if (droppedFile.size > MAX_OPML_SIZE) {
        setError('File must be smaller than 1MB')
        return
      }
      setFile(droppedFile)
      setError(null)
    } else {
      setError('Please upload an OPML file (.xml)')
    }
  }, [])

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const selected = e.target.files?.[0]
    if (selected) {
      if (selected.size > MAX_OPML_SIZE) {
        setError('File must be smaller than 1MB')
        return
      }
      setFile(selected)
      setError(null)
    }
  }

  const handleImport = async () => {
    if (!file) return

    try {
      setError(null)
      const result = await importMutation.mutateAsync(file)
      setJobId(result.job_id)
      setPhase('progress')
    } catch (err: unknown) {
      const msg = (err as Error)?.message || 'Import failed'
      setError(msg)
    }
  }

  const handleClose = () => {
    if (phase === 'done') {
      onSuccess()
    }
    setPhase('upload')
    setFile(null)
    setError(null)
    setJobId(null)
    onOpenChange(false)
  }

  const isJobDone = jobStatus?.status === 'completed' || jobStatus?.status === 'failed'

  return (
    <Dialog open={open} onOpenChange={(isOpen) => { if (!isOpen) handleClose() }}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Import OPML</DialogTitle>
          <DialogDescription>
            Import your RSS subscriptions from an OPML file.
          </DialogDescription>
        </DialogHeader>

        {phase === 'upload' && (
          <>
            <div
              className={`border-2 border-dashed rounded-lg p-8 text-center transition-colors cursor-pointer ${
                dragActive ? 'border-primary bg-primary/5' : 'border-muted-foreground/25'
              }`}
              onDragEnter={handleDrag}
              onDragLeave={handleDrag}
              onDragOver={handleDrag}
              onDrop={handleDrop}
              onClick={() => inputRef.current?.click()}
            >
              <input
                ref={inputRef}
                type="file"
                accept=".xml,.opml"
                className="hidden"
                onChange={handleFileChange}
              />
              {file ? (
                <div className="flex flex-col items-center gap-2">
                  <FileText className="h-10 w-10 text-muted-foreground" />
                  <p className="text-sm font-medium">{file.name}</p>
                  <p className="text-xs text-muted-foreground">{(file.size / 1024).toFixed(1)} KB</p>
                  <Button variant="ghost" size="sm" onClick={(e) => { e.stopPropagation(); setFile(null) }}>
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
              disabled={!file || importMutation.isPending}
              onClick={handleImport}
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

            {error && (
              <p className="text-sm text-destructive text-center">{error}</p>
            )}
          </>
        )}

        {phase === 'progress' && jobStatus && !isJobDone && (
          <div className="py-4">
            <div className="flex items-center gap-3 mb-4">
              <Loader2 className="h-5 w-5 animate-spin text-primary" />
              <span className="text-sm font-medium">
                {jobStatus.status === 'pending' ? 'Starting import...' : 'Importing feeds...'}
              </span>
            </div>
            <div className="w-full bg-muted rounded-full h-2 mb-2">
              <div
                className="bg-primary h-2 rounded-full transition-all"
                style={{ width: `${jobStatus.progress}%` }}
              />
            </div>
            <p className="text-xs text-muted-foreground text-center">
              {jobStatus.processed} / {jobStatus.total} feeds processed
              {jobStatus.failed > 0 && ` (${jobStatus.failed} failed)`}
            </p>
          </div>
        )}

        {phase === 'progress' && isJobDone && (
          <div className="py-4">
            {jobStatus!.status === 'completed' ? (
              <div className="flex flex-col items-center gap-3">
                <CheckCircle className="h-10 w-10 text-green-500" />
                <div className="text-center">
                  <p className="font-medium">Import Complete</p>
                  <p className="text-sm text-muted-foreground">
                    {jobStatus!.processed} feeds imported
                    {jobStatus!.failed > 0 && `, ${jobStatus!.failed} failed`}
                  </p>
                </div>
              </div>
            ) : (
              <div className="flex flex-col items-center gap-3">
                <AlertCircle className="h-10 w-10 text-destructive" />
                <div className="text-center">
                  <p className="font-medium">Import Failed</p>
                  <p className="text-sm text-muted-foreground">
                    {jobStatus!.error || 'Unknown error'}
                  </p>
                </div>
              </div>
            )}
            <Button className="w-full mt-4" onClick={handleClose}>
              {jobStatus!.status === 'completed' ? 'Done' : 'Close'}
            </Button>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
