import { cn } from '@/utils/cn';

type SkeletonProps = {
  className?: string;
  rounded?: 'sm' | 'md' | 'lg' | 'pill';
};

const radii = {
  sm: 'rounded-[var(--radius-sm)]',
  md: 'rounded-[var(--radius-md)]',
  lg: 'rounded-[var(--radius-lg)]',
  pill: 'rounded-[var(--radius-pill)]',
};

export function Skeleton({ className, rounded = 'md' }: SkeletonProps) {
  return (
    <span
      aria-hidden="true"
      className={cn(
        'inline-block animate-pulse bg-[color-mix(in_srgb,var(--sky-500)_12%,transparent)]',
        radii[rounded],
        className,
      )}
    />
  );
}
