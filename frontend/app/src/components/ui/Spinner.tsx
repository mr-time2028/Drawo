import { cn } from '@/utils/cn';

type SpinnerProps = {
  size?: 'sm' | 'md' | 'lg';
  className?: string;
};

const sizeMap = {
  sm: 'h-4 w-4 border-2',
  md: 'h-5 w-5 border-2',
  lg: 'h-7 w-7 border-[3px]',
} as const;

export function Spinner({ size = 'md', className }: SpinnerProps) {
  return (
    <span
      role="status"
      aria-label="Loading"
      className={cn(
        'inline-block animate-spin rounded-full border-[var(--sky-200)] border-t-[var(--sky-600)] dark:border-[rgba(142,199,255,0.25)] dark:border-t-[var(--sky-600)]',
        sizeMap[size],
        className,
      )}
    />
  );
}
