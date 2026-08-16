import { User } from 'lucide-react';
import { useState, type ImgHTMLAttributes } from 'react';

import { cn } from '@/utils/cn';

type AvatarProps = ImgHTMLAttributes<HTMLImageElement> & {
  size?: 'sm' | 'md' | 'lg' | 'xl';
  fallbackName?: string;
};

const sizeClass = {
  sm: 'drawo-avatar-sm',
  md: 'drawo-avatar-md',
  lg: 'drawo-avatar-lg',
  xl: 'drawo-avatar-xl',
};

export function Avatar({ size = 'md', src, alt = '', fallbackName, className, ...props }: AvatarProps) {
  const [errored, setErrored] = useState(false);
  const hasImage = Boolean(src) && !errored;
  const isDataUrlFallback = Boolean(fallbackName && /^data:image\//.test(fallbackName));
  // No photo → the same unknown silhouette used in the navbar. Initials are not
  // used as a stand-in for a missing avatar.
  const isUnknown = !hasImage && !isDataUrlFallback;

  return (
    <span
      className={cn('drawo-avatar', sizeClass[size], isUnknown && 'drawo-avatar-unknown', className)}
      aria-label={alt || undefined}
      aria-hidden={!alt}
    >
      {hasImage ? (
        <img src={src} alt={alt} onError={() => setErrored(true)} className="drawo-avatar-image" {...props} />
      ) : isDataUrlFallback ? (
        <img src={fallbackName} alt="" className="drawo-avatar-image" aria-hidden />
      ) : (
        <User className="drawo-avatar-icon" strokeWidth={2} aria-hidden />
      )}
    </span>
  );
}
