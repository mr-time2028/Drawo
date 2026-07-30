import { forwardRef, type InputHTMLAttributes, type TextareaHTMLAttributes } from 'react';

import { cn } from '@/utils/cn';

const baseInput = [
  'w-full rounded-[var(--radius-md)] border border-[#c8dff4] dark:border-[rgba(142,199,255,0.25)] bg-[var(--input-bg)] px-4 py-3',
  'text-[var(--ink)] placeholder:text-[color-mix(in_srgb,var(--muted)_70%,transparent)]',
  'transition-[border-color,box-shadow,background] duration-[var(--dur-fast)]',
  'focus:outline-none focus:border-[var(--sky-500)] focus:bg-[var(--card)] focus:shadow-[0_0_0_4px_rgba(74,152,247,0.16)]',
  'disabled:cursor-not-allowed disabled:opacity-60',
  'aria-[invalid=true]:border-[var(--danger)] aria-[invalid=true]:shadow-[0_0_0_4px_color-mix(in_srgb,var(--danger)_16%,transparent)]',
];

export type InputProps = InputHTMLAttributes<HTMLInputElement> & {
  invalid?: boolean;
};

export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { className, invalid, ...props },
  ref,
) {
  return (
    <input
      ref={ref}
      aria-invalid={invalid || undefined}
      className={cn(baseInput, 'h-11', className)}
      {...props}
    />
  );
});

export type TextareaProps = TextareaHTMLAttributes<HTMLTextAreaElement> & {
  invalid?: boolean;
};

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(function Textarea(
  { className, invalid, ...props },
  ref,
) {
  return (
    <textarea
      ref={ref}
      aria-invalid={invalid || undefined}
      className={cn(baseInput, 'min-h-[96px] resize-y py-3', className)}
      {...props}
    />
  );
});

export function Label({ className, htmlFor, ...props }: React.LabelHTMLAttributes<HTMLLabelElement>) {
  return (
    <label
      htmlFor={htmlFor}
      className={cn('block text-[var(--fs-sm)] font-[var(--fw-bold)] text-[var(--ink)]', className)}
      {...props}
    />
  );
}
