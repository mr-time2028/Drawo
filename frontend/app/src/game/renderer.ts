/**
 * Deterministic canvas renderer for the Drawo drawing protocol.
 *
 * Every client renders the same op log into the same logical board
 * (BOARD_WIDTH x BOARD_HEIGHT, opaque white background), so flood fills and
 * eyedropper reads produce the same result everywhere without ever shipping
 * bitmaps over the wire.
 *
 * All functions are pure with respect to the op log: replay(log) always
 * produces the same pixels. Smoothing is midpoint-quadratic (the standard
 * low-latency technique: curve through the midpoints of successive samples).
 */

import { BOARD_HEIGHT, BOARD_WIDTH, type DrawOp, type Point } from './drawTypes';

export function createBoardCanvas(): HTMLCanvasElement {
  const canvas = document.createElement('canvas');
  canvas.width = BOARD_WIDTH;
  canvas.height = BOARD_HEIGHT;
  return canvas;
}

export function clearBoard(ctx: CanvasRenderingContext2D): void {
  ctx.save();
  ctx.setTransform(1, 0, 0, 1, 0, 0);
  ctx.globalAlpha = 1;
  ctx.globalCompositeOperation = 'source-over';
  ctx.fillStyle = '#ffffff';
  ctx.fillRect(0, 0, BOARD_WIDTH, BOARD_HEIGHT);
  ctx.restore();
}

/** Replays a full op log onto a blank board. Used after undo and on sync. */
export function replay(ctx: CanvasRenderingContext2D, ops: DrawOp[]): void {
  clearBoard(ctx);
  for (const op of ops) {
    applyOp(ctx, op);
  }
}

/** Applies a single visual op (stroke/erase/shape/fill/text). */
export function applyOp(ctx: CanvasRenderingContext2D, op: DrawOp): void {
  switch (op.op) {
    case 'stroke':
      renderStroke(ctx, op);
      break;
    case 'erase':
      renderErase(ctx, op);
      break;
    case 'shape':
      renderShape(ctx, op);
      break;
    case 'fill':
      floodFill(ctx, op.x ?? 0, op.y ?? 0, op.color ?? '#000000');
      break;
    case 'text':
      renderText(ctx, op);
      break;
    default:
      // clear/undo/redo mutate the log, not the bitmap — handled by the engine.
      break;
  }
}

function opAlpha(op: DrawOp): number {
  const a = op.opacity ?? 0;
  return a > 0 && a <= 1 ? a : 1;
}

// ---------------------------------------------------------------------------
// Strokes
// ---------------------------------------------------------------------------

function tracePath(ctx: CanvasRenderingContext2D, points: Point[]): void {
  ctx.beginPath();
  ctx.moveTo(points[0].x, points[0].y);
  if (points.length === 1) {
    ctx.lineTo(points[0].x + 0.01, points[0].y);
    return;
  }
  if (points.length === 2) {
    ctx.lineTo(points[1].x, points[1].y);
    return;
  }
  for (let i = 1; i < points.length - 1; i++) {
    const midX = (points[i].x + points[i + 1].x) / 2;
    const midY = (points[i].y + points[i + 1].y) / 2;
    ctx.quadraticCurveTo(points[i].x, points[i].y, midX, midY);
  }
  const last = points[points.length - 1];
  ctx.lineTo(last.x, last.y);
}

function renderStroke(ctx: CanvasRenderingContext2D, op: DrawOp): void {
  const points = op.points ?? [];
  if (points.length === 0) return;
  const size = Math.max(1, op.size ?? 4);
  const color = op.color ?? '#000000';
  const alpha = opAlpha(op);

  ctx.save();
  ctx.lineJoin = 'round';
  ctx.lineCap = 'round';
  ctx.strokeStyle = color;
  ctx.globalCompositeOperation = 'source-over';

  switch (op.tool) {
    case 'marker': {
      // Broad translucent tip.
      ctx.globalAlpha = alpha * 0.45;
      ctx.lineWidth = size * 1.9;
      tracePath(ctx, points);
      ctx.stroke();
      break;
    }
    case 'brush': {
      // Soft edge: three concentric passes (deterministic — no shadowBlur,
      // whose rasterization differs between browsers).
      ctx.globalAlpha = alpha * 0.22;
      ctx.lineWidth = size * 1.55;
      tracePath(ctx, points);
      ctx.stroke();
      ctx.globalAlpha = alpha * 0.4;
      ctx.lineWidth = size * 1.2;
      tracePath(ctx, points);
      ctx.stroke();
      ctx.globalAlpha = alpha;
      ctx.lineWidth = size * 0.82;
      tracePath(ctx, points);
      ctx.stroke();
      break;
    }
    case 'calligraphy': {
      renderCalligraphy(ctx, points, size, color, alpha);
      break;
    }
    default: {
      // pencil — hard round line.
      ctx.globalAlpha = alpha;
      ctx.lineWidth = size;
      tracePath(ctx, points);
      ctx.stroke();
      break;
    }
  }
  ctx.restore();
}

/**
 * Calligraphy: a fixed 45° nib. Each path segment becomes a filled
 * parallelogram between nib endpoints, giving thick/thin variation with
 * direction — fully deterministic.
 */
function renderCalligraphy(
  ctx: CanvasRenderingContext2D,
  points: Point[],
  size: number,
  color: string,
  alpha: number,
): void {
  const half = Math.max(0.75, size * 0.7);
  const nx = Math.cos((-45 * Math.PI) / 180) * half;
  const ny = Math.sin((-45 * Math.PI) / 180) * half;
  ctx.globalAlpha = alpha;
  ctx.fillStyle = color;
  for (let i = 1; i < points.length; i++) {
    const a = points[i - 1];
    const b = points[i];
    ctx.beginPath();
    ctx.moveTo(a.x - nx, a.y - ny);
    ctx.lineTo(a.x + nx, a.y + ny);
    ctx.lineTo(b.x + nx, b.y + ny);
    ctx.lineTo(b.x - nx, b.y - ny);
    ctx.closePath();
    ctx.fill();
  }
}

function renderErase(ctx: CanvasRenderingContext2D, op: DrawOp): void {
  const points = op.points ?? [];
  if (points.length === 0) return;
  ctx.save();
  ctx.lineJoin = 'round';
  ctx.lineCap = 'round';
  ctx.globalAlpha = 1;
  ctx.globalCompositeOperation = 'source-over';
  ctx.strokeStyle = '#ffffff';
  ctx.lineWidth = Math.max(1, op.size ?? 20);
  tracePath(ctx, points.length === 1 ? [points[0], points[0]] : points);
  ctx.stroke();
  ctx.restore();
}

// ---------------------------------------------------------------------------
// Shapes
// ---------------------------------------------------------------------------

function renderShape(ctx: CanvasRenderingContext2D, op: DrawOp): void {
  const color = op.color ?? '#000000';
  const lineWidth = Math.max(1, op.size ?? 4);
  const x = op.x ?? 0;
  const y = op.y ?? 0;
  const w = op.width ?? 0;
  const h = op.height ?? 0;

  ctx.save();
  ctx.globalAlpha = opAlpha(op);
  ctx.globalCompositeOperation = 'source-over';
  ctx.strokeStyle = color;
  ctx.fillStyle = color;
  ctx.lineWidth = lineWidth;
  ctx.lineJoin = 'round';
  ctx.lineCap = 'round';

  switch (op.shape) {
    case 'line': {
      ctx.beginPath();
      ctx.moveTo(x, y);
      ctx.lineTo(w, h);
      ctx.stroke();
      break;
    }
    case 'arrow': {
      renderArrow(ctx, x, y, w, h, lineWidth);
      break;
    }
    case 'rectangle': {
      ctx.beginPath();
      ctx.rect(x, y, w, h);
      if (op.filled) ctx.fill();
      else ctx.stroke();
      break;
    }
    case 'ellipse': {
      ctx.beginPath();
      ctx.ellipse(x + w / 2, y + h / 2, w / 2, h / 2, 0, 0, Math.PI * 2);
      if (op.filled) ctx.fill();
      else ctx.stroke();
      break;
    }
    case 'triangle': {
      ctx.beginPath();
      ctx.moveTo(x + w / 2, y);
      ctx.lineTo(x + w, y + h);
      ctx.lineTo(x, y + h);
      ctx.closePath();
      if (op.filled) ctx.fill();
      else ctx.stroke();
      break;
    }
    default:
      break;
  }
  ctx.restore();
}

function renderArrow(
  ctx: CanvasRenderingContext2D,
  x1: number,
  y1: number,
  x2: number,
  y2: number,
  lineWidth: number,
): void {
  const angle = Math.atan2(y2 - y1, x2 - x1);
  const headLen = Math.max(10, lineWidth * 3.5);
  // Shorten the shaft so it doesn't poke through the head tip.
  const shaftX = x2 - Math.cos(angle) * headLen * 0.6;
  const shaftY = y2 - Math.sin(angle) * headLen * 0.6;

  ctx.beginPath();
  ctx.moveTo(x1, y1);
  ctx.lineTo(shaftX, shaftY);
  ctx.stroke();

  ctx.beginPath();
  ctx.moveTo(x2, y2);
  ctx.lineTo(x2 - headLen * Math.cos(angle - Math.PI / 6), y2 - headLen * Math.sin(angle - Math.PI / 6));
  ctx.lineTo(x2 - headLen * Math.cos(angle + Math.PI / 6), y2 - headLen * Math.sin(angle + Math.PI / 6));
  ctx.closePath();
  ctx.fill();
}

// ---------------------------------------------------------------------------
// Text
// ---------------------------------------------------------------------------

function renderText(ctx: CanvasRenderingContext2D, op: DrawOp): void {
  const text = op.text ?? '';
  if (!text) return;
  ctx.save();
  ctx.globalAlpha = opAlpha(op);
  ctx.globalCompositeOperation = 'source-over';
  ctx.fillStyle = op.color ?? '#000000';
  const size = Math.max(8, Math.min(128, op.font_size ?? 24));
  ctx.font = `${size}px Arial, "Segoe UI", Tahoma, sans-serif`;
  ctx.textBaseline = 'top';
  ctx.fillText(text, op.x ?? 0, op.y ?? 0);
  ctx.restore();
}

// ---------------------------------------------------------------------------
// Flood fill (scanline BFS with tolerance, on the fixed logical bitmap)
// ---------------------------------------------------------------------------

const FILL_TOLERANCE_SQ = 40 * 40;

function hexToRGBA(hex: string): [number, number, number, number] {
  const n = parseInt(hex.slice(1), 16);
  return [(n >> 16) & 0xff, (n >> 8) & 0xff, n & 0xff, 255];
}

export function floodFill(ctx: CanvasRenderingContext2D, fx: number, fy: number, hexColor: string): void {
  const startX = Math.round(fx);
  const startY = Math.round(fy);
  if (startX < 0 || startY < 0 || startX >= BOARD_WIDTH || startY >= BOARD_HEIGHT) return;

  const img = ctx.getImageData(0, 0, BOARD_WIDTH, BOARD_HEIGHT);
  const data = img.data;
  const [fr, fg, fb, fa] = hexToRGBA(hexColor);

  const startIdx = (startY * BOARD_WIDTH + startX) * 4;
  const sr = data[startIdx];
  const sg = data[startIdx + 1];
  const sb = data[startIdx + 2];

  // Filling with (almost) the target color is a no-op; avoids infinite loops.
  const dSame = (sr - fr) ** 2 + (sg - fg) ** 2 + (sb - fb) ** 2;
  if (dSame <= 16) return;

  const matches = (idx: number): boolean => {
    const dr = data[idx] - sr;
    const dg = data[idx + 1] - sg;
    const db = data[idx + 2] - sb;
    return dr * dr + dg * dg + db * db <= FILL_TOLERANCE_SQ;
  };
  const paint = (idx: number): void => {
    data[idx] = fr;
    data[idx + 1] = fg;
    data[idx + 2] = fb;
    data[idx + 3] = fa;
  };

  // Scanline stack fill.
  const stack: number[] = [startX, startY];
  while (stack.length > 0) {
    const y = stack.pop() as number;
    let x = stack.pop() as number;

    let idx = (y * BOARD_WIDTH + x) * 4;
    if (!matches(idx)) continue;

    // Walk left to the span start.
    while (x > 0 && matches(idx - 4)) {
      x--;
      idx -= 4;
    }

    let spanUp = false;
    let spanDown = false;
    while (x < BOARD_WIDTH && matches(idx)) {
      paint(idx);
      if (y > 0) {
        const upIdx = idx - BOARD_WIDTH * 4;
        if (matches(upIdx)) {
          if (!spanUp) {
            stack.push(x, y - 1);
            spanUp = true;
          }
        } else {
          spanUp = false;
        }
      }
      if (y < BOARD_HEIGHT - 1) {
        const downIdx = idx + BOARD_WIDTH * 4;
        if (matches(downIdx)) {
          if (!spanDown) {
            stack.push(x, y + 1);
            spanDown = true;
          }
        } else {
          spanDown = false;
        }
      }
      x++;
      idx += 4;
    }
  }

  ctx.putImageData(img, 0, 0);
}

/** Reads a single board pixel as #rrggbb (eyedropper). */
export function pickColor(ctx: CanvasRenderingContext2D, x: number, y: number): string {
  const px = Math.min(BOARD_WIDTH - 1, Math.max(0, Math.round(x)));
  const py = Math.min(BOARD_HEIGHT - 1, Math.max(0, Math.round(y)));
  const d = ctx.getImageData(px, py, 1, 1).data;
  return `#${((1 << 24) | (d[0] << 16) | (d[1] << 8) | d[2]).toString(16).slice(1)}`;
}
