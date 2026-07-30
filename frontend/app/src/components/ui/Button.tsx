import { cva, type VariantProps } from 'class-variance-authority';
import { forwardRef, type ButtonHTMLAttributes } from 'react';

import { Spinner } from './Spinner';
import { cn } from '@/utils/cn';

const buttonVariants = cva(
  [
    'inline-flex items-center justify-center gap-2 select-none',
    'font-black tracking-tight',
    'transition-[transform,background-color,color,box-shadow,opacity] duration-[var(--dur-fast)] ease-[var(--ease-out)]',
    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--sky-500)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--card-solid)]',
    'disabled:cursor-not-allowed disabled:opacity-60 disabled:pointer-events-none',
  ],
  {
    variants: {
      variant: {
        primary:
          'bg-[var(--sky-500)] text-white shadow-[0_16px_34px_rgba(74,152,247,0.32)] hover:bg-[var(--sky-600)] active:translate-y-[1px]',
        secondary:
          'bg-[var(--sky-100)] text-[var(--sky-600)] hover:bg-[var(--sky-200)] dark:bg-[color-mix(in_srgb,var(--sky-500)_18%,transparent)] dark:text-[var(--sky-600)] dark:hover:bg-[color-mix(in_srgb,var(--sky-500)_28%,transparent)]',
        ghost:
          'bg-transparent text-[var(--sky-600)] hover:bg-[color-mix(in_srgb,var(--sky-500)_12%,transparent)] dark:text-[var(--sky-600)]',
        danger:
          'bg-[var(--danger)] text-white shadow-[0_12px_24px_rgba(220,38,38,0.28)] hover:brightness-110 active:translate-y-[1px]',
        outline:
          'border border-[var(--border-strong)] bg-[var(--card)] text-[var(--ink)] hover:bg-[var(--sky-50)] dark:hover:bg-[var(--sky-100)]',
      },
      size: {
        sm: 'h-9 min-h-9 px-3.5 text-[var(--fs-sm)] rounded-[var(--radius-pill)]',
        md: 'h-11 min-h-11 px-5 text-[var(--fs-md)] rounded-[var(--radius-pill)]',
        lg: 'h-12 min-h-12 px-6 text-[var(--fs-lg)] rounded-[var(--radius-pill)]',
        icon: 'h-10 w-10 rounded-[var(--radius-pill)]',
      },
      fullWidth: {
        true: 'w-full',
        false: '',
      },
    },
    defaultVariants: {
      variant: 'primary',
      size: 'md',
      fullWidth: false,
    },
  },
);

export type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> &
  VariantProps<typeof buttonVariants> & {
    loading?: boolean;
    leftIcon?: React.ReactNode;
    rightIcon?: React.ReactNode;
  };

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { className, variant, size, fullWidth, loading, disabled, children, leftIcon, rightIcon, ...props },
  ref,
) {
  return (
    <button
      ref={ref}
      className={cn(buttonVariants({ variant, size, fullWidth }), className)}
      disabled={disabled || loading}
      {...props}
    >
      {loading ? <Spinner size="sm" /> : leftIcon}
      {children}
      {!loading && rightIcon}
    </button>
  );
});
