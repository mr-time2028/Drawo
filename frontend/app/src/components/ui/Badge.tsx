import { cva, type VariantProps } from 'class-variance-authority';

import { cn } from '@/utils/cn';

const badgeVariants = cva(
  'inline-flex items-center gap-1 rounded-[var(--radius-pill)] px-2.5 py-1 text-[var(--fs-xs)] font-[var(--fw-bold)] leading-none',
  {
    variants: {
      variant: {
        default:
          'bg-[var(--sky-100)] text-[var(--sky-600)] dark:bg-[color-mix(in_srgb,var(--sky-500)_20%,transparent)]',
        success:
          'bg-[var(--success-bg)] text-[var(--success)] dark:bg-[color-mix(in_srgb,var(--success)_20%,transparent)]',
        danger:
          'bg-[var(--danger-bg)] text-[var(--danger)] dark:bg-[color-mix(in_srgb,var(--danger)_20%,transparent)]',
        warning:
          'bg-[var(--warning-bg)] text-[var(--warning)] dark:bg-[color-mix(in_srgb,var(--warning)_20%,transparent)]',
        neutral: 'bg-[color-mix(in_srgb,var(--muted)_14%,transparent)] text-[var(--muted)]',
      },
    },
    defaultVariants: { variant: 'default' },
  },
);

export type BadgeProps = React.HTMLAttributes<HTMLSpanElement> & VariantProps<typeof badgeVariants>;

export function Badge({ className, variant, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ variant }), className)} {...props} />;
}
