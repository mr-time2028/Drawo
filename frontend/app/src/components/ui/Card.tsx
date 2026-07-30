import { cva, type VariantProps } from 'class-variance-authority';
import { forwardRef, type HTMLAttributes } from 'react';

import { cn } from '@/utils/cn';

const cardVariants = cva(
  [
    'relative rounded-[var(--radius-xl)] border border-[var(--border)] bg-[var(--card)]',
    'shadow-[var(--elev-2)] backdrop-blur-[18px] transition-shadow duration-[var(--dur-base)]',
  ],
  {
    variants: {
      tone: {
        default: '',
        ghost: 'bg-transparent shadow-none',
        hero: 'rounded-[var(--radius-2xl)]',
      },
      padding: {
        none: '',
        sm: 'p-4',
        md: 'p-6',
        lg: 'p-8',
      },
      hoverable: {
        true: 'hover:shadow-[var(--elev-3)] hover:-translate-y-0.5 cursor-pointer',
        false: '',
      },
    },
    defaultVariants: {
      tone: 'default',
      padding: 'md',
      hoverable: false,
    },
  },
);

export type CardProps = HTMLAttributes<HTMLDivElement> & VariantProps<typeof cardVariants>;

export const Card = forwardRef<HTMLDivElement, CardProps>(function Card(
  { className, tone, padding, hoverable, ...props },
  ref,
) {
  return <div ref={ref} className={cn(cardVariants({ tone, padding, hoverable }), className)} {...props} />;
});

export function CardHeader({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('mb-4 flex items-start justify-between gap-3', className)} {...props} />;
}

export function CardTitle({ className, children, ...props }: HTMLAttributes<HTMLHeadingElement>) {
  return (
    <h3
      className={cn('text-[var(--fs-xl)] font-[var(--fw-black)] leading-tight text-[var(--ink)]', className)}
      {...props}
    >
      {children}
    </h3>
  );
}

export function CardDescription({ className, ...props }: HTMLAttributes<HTMLParagraphElement>) {
  return <p className={cn('text-[var(--fs-sm)] text-[var(--muted)]', className)} {...props} />;
}
