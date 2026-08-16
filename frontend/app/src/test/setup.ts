import '@testing-library/jest-dom/vitest';
import { afterEach, vi } from 'vitest';

import { i18n } from '../i18n';
import { resetAuthStore } from '../stores/authStore';
import { resetThemeStore } from '../stores/themeStore';

Object.defineProperty(window, 'scrollTo', {
  value: vi.fn(),
  writable: true,
});

// jsdom implements neither smooth scrolling helpers nor pointer capture.
if (typeof Element !== 'undefined') {
  if (!Element.prototype.scrollIntoView) {
    Element.prototype.scrollIntoView = () => {};
  }
  if (!Element.prototype.setPointerCapture) {
    Element.prototype.setPointerCapture = () => {};
    Element.prototype.releasePointerCapture = () => {};
    Element.prototype.hasPointerCapture = () => false;
  }
}

// jsdom ships no 2D canvas raster. The game board engine only needs the API
// surface (the op-log logic is what tests assert), so provide a no-op context.
const noopCtx = () => {
  const imageData = (w: number, h: number) => ({
    data: new Uint8ClampedArray(w * h * 4),
    width: w,
    height: h,
  });
  return {
    canvas: null as unknown,
    save: vi.fn(),
    restore: vi.fn(),
    setTransform: vi.fn(),
    clearRect: vi.fn(),
    fillRect: vi.fn(),
    beginPath: vi.fn(),
    closePath: vi.fn(),
    moveTo: vi.fn(),
    lineTo: vi.fn(),
    quadraticCurveTo: vi.fn(),
    ellipse: vi.fn(),
    rect: vi.fn(),
    stroke: vi.fn(),
    fill: vi.fn(),
    fillText: vi.fn(),
    drawImage: vi.fn(),
    getImageData: vi.fn((_x: number, _y: number, w: number, h: number) => imageData(w, h)),
    putImageData: vi.fn(),
    globalAlpha: 1,
    globalCompositeOperation: 'source-over',
    fillStyle: '',
    strokeStyle: '',
    lineWidth: 1,
    lineCap: 'round',
    lineJoin: 'round',
    font: '',
    textBaseline: 'top',
  } as unknown as CanvasRenderingContext2D;
};

if (typeof HTMLCanvasElement !== 'undefined') {
  // Plain function (not vi.fn) so vi.restoreAllMocks() cannot strip it.
  HTMLCanvasElement.prototype.getContext = function getContext() {
    return noopCtx();
  } as unknown as typeof HTMLCanvasElement.prototype.getContext;
}

afterEach(() => {
  localStorage.clear();
  sessionStorage.clear();
  resetAuthStore();
  resetThemeStore();
  void i18n.changeLanguage('fa');
  vi.restoreAllMocks();
});
