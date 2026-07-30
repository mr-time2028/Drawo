import { User } from 'lucide-react';
import { useState, type ImgHTMLAttributes } from 'react';

import { cn } from '@/utils/cn';

type AvatarProps = ImgHTMLAttributes<HTMLImageElement> & {
  size?: 'sm' | 'md' | 'lg' | 'xl';
  fallbackName?: string;
};

const sizeMap = {
  sm: 'h-9 w-9 text-[var(--fs-sm)]',
  md: 'h-11 w-11 text-[var(--fs-md)]',
  lg: 'h-14 w-14 text-[var(--fs-lg)]',
  xl: 'h-20 w-20 text-[var(--fs-2xl)]',
};

function initials(name?: string) {
  if (!name) return '';
  const trimmed = name.trim();
  if (!trimmed) return '';
  const parts = trimmed.split(/\s+/);
  return parts.length > 1 ? (parts[0][0] + parts[1][0]).toUpperCase() : parts[0][0].toUpperCase();
}

function colorFor(name: string) {
  const colors = ['#4A98F7', '#22C55E', '#F97316', '#EF4444', '#A855F7', '#14B8A6', '#F59E0B', '#EC4899'];
  const idx = [...name].reduce((acc, ch) => acc + ch.charCodeAt(0), 0) % colors.length;
  return colors[idx];
}

export function Avatar({ size = 'md', src, alt = '', fallbackName, className, ...props }: AvatarProps) {
  const [errored, setErrored] = useState(false);
  const hasImage = Boolean(src) && !errored;
  const isDataUrlFallback = Boolean(fallbackName && /^data:image\//.test(fallbackName));
  const hasInitials = Boolean(initials(fallbackName || alt)) && !isDataUrlFallback;
  // Unknown-user fallback (no name, no image, no data URL): neutral black/white
  // silhouette icon. No fancy color, no animation, no letter.
  const isUnknown = !hasImage && !isDataUrlFallback && !hasInitials;

  const letter = initials(fallbackName || alt);
  const nameForColor = (fallbackName || alt || '?').toString();
  const bg = colorFor(nameForColor);

  return (
    <span
      className={cn(
        'relative inline-flex shrink-0 items-center justify-center overflow-hidden rounded-full',
        sizeMap[size],
        // Unknown-user fallback is intentionally transparent & borderless so the
        // icon blends into whatever container (e.g. the navbar avatar button)
        // provides the circular background. Apply bg/border via `className` if
        // you need a standalone filled avatar.
        isUnknown && 'bg-transparent text-[var(--ink)]',
        hasInitials && 'font-extrabold text-white',
        className,
      )}
      style={
        hasImage || isDataUrlFallback || isUnknown
          ? hasInitials
            ? { background: bg }
            : undefined
          : { background: bg, color: '#fff' }
      }
      aria-label={alt}
    >
      {hasImage ? (
        <img
          src={src}
          alt={alt}
          onError={() => setErrored(true)}
          className="h-full w-full object-cover"
          {...props}
        />
      ) : isDataUrlFallback ? (
        <img src={fallbackName} alt="" className="h-full w-full object-cover" aria-hidden />
      ) : hasInitials ? (
        <span aria-hidden>{letter}</span>
      ) : (
        <User className="h-1/2 w-1/2" strokeWidth={2} aria-hidden />
      )}
    </span>
  );
}
