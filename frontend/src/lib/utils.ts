import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatDate(dateStr?: string | null): string {
  if (!dateStr) return '—'
  try {
    return new Date(dateStr).toLocaleDateString('mn-MN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
    })
  } catch {
    return dateStr
  }
}

export function formatArea(m2?: number | null): string {
  if (m2 == null) return '—'
  if (m2 >= 10000) return `${(m2 / 10000).toFixed(2)} га`
  return `${m2.toFixed(1)} м²`
}
