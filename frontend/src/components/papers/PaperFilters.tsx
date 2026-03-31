import { ArrowUpDown, Filter } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from '@/components/ui/dropdown-menu'
import type { PaperStatus } from '@/types/paper'

interface PaperFiltersProps {
  status: PaperStatus | ''
  onStatusChange: (status: PaperStatus | '') => void
  year: string
  onYearChange: (year: string) => void
  sort: string
  order: 'asc' | 'desc'
  onSortChange: (sort: string, order: 'asc' | 'desc') => void
}

const statusOptions: { value: PaperStatus | ''; label: string }[] = [
  { value: '', label: 'All Status' },
  { value: 'completed', label: 'Completed' },
  { value: 'processing', label: 'Processing' },
  { value: 'pending', label: 'Pending' },
  { value: 'failed', label: 'Failed' },
]

const sortOptions: { sort: string; order: 'asc' | 'desc'; label: string }[] = [
  { sort: 'created_at', order: 'desc', label: 'Newest' },
  { sort: 'created_at', order: 'asc', label: 'Oldest' },
  { sort: 'title', order: 'asc', label: 'Title A-Z' },
  { sort: 'title', order: 'desc', label: 'Title Z-A' },
  { sort: 'published_year', order: 'desc', label: 'Year Desc' },
  { sort: 'published_year', order: 'asc', label: 'Year Asc' },
]

export function PaperFilters({
  status,
  onStatusChange,
  year,
  onYearChange,
  sort,
  order,
  onSortChange,
}: PaperFiltersProps) {
  const currentSortLabel =
    sortOptions.find((o) => o.sort === sort && o.order === order)?.label || 'Newest'

  return (
    <div className="flex items-center gap-2 flex-wrap">
      {/* Status Filter */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm" className="gap-1">
            <Filter className="h-3 w-3" />
            {status ? statusOptions.find((o) => o.value === status)?.label : 'All Status'}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start">
          {statusOptions.map((opt) => (
            <DropdownMenuItem
              key={opt.value}
              onClick={() => onStatusChange(opt.value)}
            >
              {opt.label}
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>

      {/* Year Filter */}
      <Input
        type="number"
        placeholder="Year"
        min="1900"
        max={new Date().getFullYear()}
        value={year}
        onChange={(e) => onYearChange(e.target.value)}
        className="w-24 h-9 text-sm"
      />

      {/* Sort */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm" className="gap-1">
            <ArrowUpDown className="h-3 w-3" />
            {currentSortLabel}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          {sortOptions.map((opt) => (
            <DropdownMenuItem
              key={`${opt.sort}-${opt.order}`}
              onClick={() => onSortChange(opt.sort, opt.order)}
            >
              {opt.label}
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}
