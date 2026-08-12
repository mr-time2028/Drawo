import { describe, expect, it } from 'vitest';

import { safeNextPath } from './redirect';

describe('safeNextPath', () => {
  it('accepts clean same-origin paths', () => {
    expect(safeNextPath('/app')).toBe('/app');
    expect(safeNextPath('/r/ABCDEF')).toBe('/r/ABCDEF');
    expect(safeNextPath('/rooms/x?a=1&b=2')).toBe('/rooms/x?a=1&b=2');
  });

  it('trims surrounding whitespace', () => {
    expect(safeNextPath('  /app  ')).toBe('/app');
  });

  it('rejects empty / missing / non-strings', () => {
    expect(safeNextPath('')).toBeNull();
    expect(safeNextPath('   ')).toBeNull();
    expect(safeNextPath(undefined)).toBeNull();
    expect(safeNextPath(null)).toBeNull();
    expect(safeNextPath(123)).toBeNull();
    expect(safeNextPath({})).toBeNull();
  });

  it('rejects protocol-relative URLs (open-redirect)', () => {
    expect(safeNextPath('//evil.com')).toBeNull();
  });

  it('rejects absolute URLs', () => {
    expect(safeNextPath('https://evil.com')).toBeNull();
    expect(safeNextPath('http://evil.com/phish')).toBeNull();
    expect(safeNextPath('javascript:alert(1)')).toBeNull();
  });

  it('allows deep paths with fragments and percent-encoding', () => {
    expect(safeNextPath('/rooms/abc#players')).toBe('/rooms/abc#players');
    expect(safeNextPath('/r/ABC%20DEF')).toBe('/r/ABC%20DEF');
  });
});
