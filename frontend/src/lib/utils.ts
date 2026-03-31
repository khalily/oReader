import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function parseJsonArray(str: string | null): string[] {
  if (!str) return []
  try {
    const result = JSON.parse(str)
    return Array.isArray(result) ? result : []
  } catch { return [] }
}
