import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

/**
 * Combine class names with both conditional support (`clsx`) and tailwind-class
 * deduplication (`tailwind-merge`). Even though we aren't using Tailwind today,
 * twMerge is a harmless post-processor that resolves conflicting utility-style
 * classes and keeps CVA output clean. Add custom tailwind-like tokens here later
 * if needed.
 */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}
