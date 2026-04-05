import { Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useAuthStore } from '@/stores/authStore'

interface SidebarHeaderProps {
  onAddClick: () => void
}

export function SidebarHeader({ onAddClick }: SidebarHeaderProps) {
  const user = useAuthStore((s) => s.user)
  const initial = user?.nickname?.[0] || user?.email?.[0] || '?'

  return (
    <div className="flex items-center justify-between px-4 py-3 border-b">
      <div className="flex items-center gap-2">
        <div className="w-6 h-6 rounded-full bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center">
          <span className="text-white text-xs font-bold">o</span>
        </div>
        <span className="font-semibold text-sm">oReader</span>
      </div>
      <div className="flex items-center gap-1">
        <Button variant="ghost" size="sm" className="h-7 px-2 text-xs" onClick={onAddClick}>
          <Plus className="h-3.5 w-3.5 mr-1" />
          新增
        </Button>
        <div className="w-7 h-7 rounded-full bg-gradient-to-br from-emerald-400 to-cyan-500 flex items-center justify-center ml-1">
          <span className="text-white text-xs font-medium">{initial.toUpperCase()}</span>
        </div>
      </div>
    </div>
  )
}
