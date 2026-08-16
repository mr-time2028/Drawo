/**
 * Drawing protocol types — the TypeScript mirror of
 * backend/app/internal/realtime/drawing.go.
 *
 * Coordinates are in BOARD SPACE: a fixed logical canvas of
 * BOARD_WIDTH x BOARD_HEIGHT. Every client renders ops into the same logical
 * space and scales to its own screen, which keeps fills and strokes
 * deterministic across devices and matches the server's coordinate
 * validation (0..4096).
 */

export const BOARD_WIDTH = 1024;
export const BOARD_HEIGHT = 768;

/** Server-enforced limits (keep in sync with drawing.go). */
export const MAX_STROKE_POINTS = 256;
export const MAX_BRUSH_SIZE = 64;
export const MAX_TEXT_LENGTH = 60;
export const MAX_DRAW_OPS_PER_SECOND = 30;

export type StrokeTool = 'pencil' | 'brush' | 'marker' | 'calligraphy';
export type ShapeKind = 'line' | 'rectangle' | 'ellipse' | 'triangle' | 'arrow';

export type Point = { x: number; y: number };

export type DrawOp = {
  op: 'stroke' | 'erase' | 'shape' | 'fill' | 'text' | 'clear' | 'undo' | 'redo';

  // Server-authoritative metadata (present on broadcast frames).
  id?: string;
  user_id?: string;
  server_seq?: number;
  timestamp?: number;

  // Undo/redo metadata.
  target_id?: string;
  target?: DrawOp;

  // Stroke/erase.
  tool?: StrokeTool;
  color?: string;
  size?: number;
  points?: Point[];
  opacity?: number;

  // Shape/fill. For line/arrow: (x,y) start, (width,height) end point.
  shape?: ShapeKind;
  x?: number;
  y?: number;
  width?: number;
  height?: number;
  filled?: boolean;

  // Text.
  text?: string;
  font_size?: number;
};

export type CanvasSyncPayload = {
  operations: DrawOp[];
  server_seq: number;
};

/** UI tool identifiers (superset of wire tools). */
export type ToolId = StrokeTool | 'eraser' | 'fill' | 'eyedropper' | 'text' | 'pan' | `shape:${ShapeKind}`;

export function isStrokeTool(t: ToolId): t is StrokeTool {
  return t === 'pencil' || t === 'brush' || t === 'marker' || t === 'calligraphy';
}

export function clampBoardPoint(p: Point): Point {
  return {
    x: Math.min(BOARD_WIDTH, Math.max(0, p.x)),
    y: Math.min(BOARD_HEIGHT, Math.max(0, p.y)),
  };
}

/** Round to 1 decimal — enough precision, ~35% smaller JSON payloads. */
export function compactPoint(p: Point): Point {
  return { x: Math.round(p.x * 10) / 10, y: Math.round(p.y * 10) / 10 };
}
