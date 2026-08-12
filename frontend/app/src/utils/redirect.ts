const SAFE_NEXT_RE = /^\/[a-zA-Z0-9\-_/?=&.%@:+~#]*$/;

/**
 * Validate a `?next=` redirect target to prevent open-redirect (phishing)
 * attacks: only same-origin, path-absolute URLs are allowed. Returns the
 * sanitized path or `null` when the input is unsafe/missing.
 */
export function safeNextPath(next: unknown): string | null {
  if (typeof next !== 'string') return null;
  const trimmed = next.trim();
  if (!trimmed) return null;
  if (!SAFE_NEXT_RE.test(trimmed)) return null;
  // Protocol-relative URLs (//evil.com) start with "//" — reject them.
  if (trimmed.startsWith('//')) return null;
  return trimmed;
}
