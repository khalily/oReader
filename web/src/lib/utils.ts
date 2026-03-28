import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function parseJsonArray(str: string | null): string[] {
  if (!str) return []
  try { return JSON.parse(str) } catch { return [] }
}
