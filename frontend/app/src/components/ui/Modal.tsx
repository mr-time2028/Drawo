import { X } from 'lucide-react';
import { useEffect, type HTMLAttributes, type ReactNode } from 'react';
import { createPortal } from 'react-dom';

import { cn } from '@/utils/cn';

type ModalProps = {
  open: boolean;
  onClose: () => void;
  title?: ReactNode;
  description?: ReactNode;
  children?: ReactNode;
  /** Optional footer rendered outside the scrollable body (sticky at the
   *  bottom of the panel). Use <ModalFooter> for consistent styling. */
  footer?: ReactNode;
  className?: string;
  /** Label used by screen readers when no title is provided. */
  ariaLabel?: string;
};

export function Modal({
  open,
  onClose,
  title,
  description,
  children,
  footer,
  className,
  ariaLabel,
}: ModalProps) {
  // Close on Escape. We listen in the capture phase and stopPropagation so
  // that when multiple modals/drawers are mounted (e.g. StartMatchDrawer is
  // still in the tree while PrivateRoomModal opens) the Escape only closes
  // the topmost one and doesn't also dismiss siblings/ancestors.
  useEffect(() => {
    if (!open) return;
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        e.stopPropagation();
        onClose();
      }
    }
    window.addEventListener('keydown', onKey, true);
    // Prevent page scroll while modal is open
    const prev = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      window.removeEventListener('keydown', onKey, true);
      document.body.style.overflow = prev;
    };
  }, [open, onClose]);

  if (!open) return null;

  return createPortal(
    <div
      className="drawo-modal-root"
      role="dialog"
      aria-modal="true"
      aria-label={title ? undefined : ariaLabel}
      aria-labelledby={title ? 'drawo-modal-title' : undefined}
    >
      <div className="drawo-modal-backdrop" onClick={onClose} aria-hidden="true" />
      <div className="drawo-modal-panel" role="document">
        <div className={cn('drawo-modal-content', className)}>
          {(title || description) && (
            <div className="drawo-modal-header">
              <div className="drawo-modal-heading">
                {title && (
                  <h2 id="drawo-modal-title" className="drawo-modal-title">
                    {title}
                  </h2>
                )}
                {description && <p className="drawo-modal-description">{description}</p>}
              </div>
              <button type="button" className="drawo-modal-close" aria-label="Close" onClick={onClose}>
                <X size={20} strokeWidth={2.4} aria-hidden="true" />
              </button>
            </div>
          )}
          <div className="drawo-modal-body">{children}</div>
          {footer && <div className="drawo-modal-footer">{footer}</div>}
        </div>
      </div>
    </div>,
    document.body,
  );
}

export function ModalFooter({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('drawo-modal-footer-actions', className)} {...props} />;
}
