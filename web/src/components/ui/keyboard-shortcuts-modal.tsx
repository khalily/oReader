import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { DEFAULT_SHORTCUTS } from '@/hooks/useKeyboardShortcuts'

interface KeyboardShortcutsModalProps {
  isOpen: boolean
  onClose: () => void
  shortcuts?: readonly { key: string; description: string }[]
}

/**
 * Modal displaying all available keyboard shortcuts
 */
export function KeyboardShortcutsModal({
  isOpen,
  onClose,
  shortcuts = DEFAULT_SHORTCUTS,
}: KeyboardShortcutsModalProps) {
  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Keyboard Shortcuts</DialogTitle>
          <DialogDescription>
            Navigate and control oReader without touching the mouse
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-1">
          {shortcuts.map((shortcut) => (
            <div
              key={shortcut.key}
              className="flex items-center justify-between py-2 px-3 rounded-md hover:bg-accent"
            >
              <span className="text-sm text-muted-foreground">
                {shortcut.description}
              </span>
              <kbd className="px-2 py-1 text-xs font-semibold text-foreground bg-muted border border-border rounded-md min-w-[40px] text-center">
                {shortcut.key}
              </kbd>
            </div>
          ))}
        </div>

        <div className="flex justify-end pt-4">
          <Button onClick={onClose}>Close</Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}

/**
 * Card variant of keyboard shortcuts (for use in settings or about pages)
 */
export function KeyboardShortcutsCard({
  shortcuts = DEFAULT_SHORTCUTS,
}: {
  shortcuts?: readonly { key: string; description: string }[]
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Keyboard Shortcuts</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-2 gap-2">
          {shortcuts.map((shortcut) => (
            <div
              key={shortcut.key}
              className="flex items-center justify-between text-sm"
            >
              <span className="text-muted-foreground">
                {shortcut.description}
              </span>
              <kbd className="px-2 py-1 text-xs font-semibold bg-muted border rounded">
                {shortcut.key}
              </kbd>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
